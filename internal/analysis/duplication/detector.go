package duplication

import (
	"maps"
	"slices"
	"strings"

	"github.com/zeebo/xxh3"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/analysis"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/fingerprint"
)

// occurrence is one copy of a clone region. It is serialised as evidence.
type occurrence struct {
	File       string `json:"file"`
	Symbol     string `json:"symbol,omitempty"`
	StartIndex int    `json:"startIndex"`
	EndIndex   int    `json:"endIndex"`
	StartLine  int    `json:"startLine"`
	EndLine    int    `json:"endLine"`
}

// cloneGroup is one maximal region of tokens that appears, identically, in two
// or more places. Occurrences are in file path order, then by position.
type cloneGroup struct {
	Fingerprint string
	Tokens      int
	Occurrences []occurrence
}

// windowKey is a 128-bit rolling hash that only preselects windows. Groups use the
// SHA-256 fingerprint, so a collision costs a hash but never merges clones.
type windowKey [2]uint64

// Two independent polynomial bases give the pair its 128 bits.
const (
	rollingBase1 = 0x100000001b3
	rollingBase2 = 0x9e3779b97f4a7c15
)

// run is a range of token indexes, [start, end), that no import declaration
// interrupts. A window never crosses a run boundary.
type run struct {
	start, end int
}

// functionSpan names the function that owns a range of lines.
type functionSpan struct {
	symbol        string
	line, endLine int
}

// source is one file prepared for detection.
type source struct {
	file      *adapter.SourceFile
	hashes    []uint64
	runs      []run
	runEnd    []int // per token: exclusive end of its run, 0 for a skipped token
	functions []functionSpan
	repeated  []string // per token: fingerprint of the repeated window starting there
	covered   []bool   // per token: already inside a reported region
}

// window is a candidate copy: minTokens tokens starting at one position.
type window struct {
	src   *source
	start int
}

// detector finds exact repeated token regions.
type detector struct {
	minTokens int
	minLines  int
}

func newDetector(minTokens, minLines int) detector {
	return detector{minTokens: minTokens, minLines: minLines}
}

// detect returns clone groups in file order and the duplicated lines per file.
// Every repeated window counts toward the duplicated lines; a group is the
// longest region its copies share, seeded at the first uncovered repeated
// window in file path order, so unrelated edits do not change which regions
// are reported.
func (d detector) detect(files []*adapter.SourceFile) ([]cloneGroup, map[string]analysis.LineSet) {
	sources := d.prepare(files)
	windows := d.repeatedWindows(sources)

	duplicated := make(map[string]analysis.LineSet, len(files))
	for _, file := range files {
		duplicated[file.RelativePath] = analysis.LineSet{}
	}
	for _, fp := range slices.Sorted(maps.Keys(windows)) {
		for _, w := range windows[fp] {
			w.src.repeated[w.start] = fp
			lines := duplicated[w.src.file.RelativePath]
			for line := w.line(); line <= w.endLine(d.minTokens); line++ {
				lines.Add(line)
			}
		}
	}

	var groups []cloneGroup
	for _, src := range sources {
		for start, fp := range src.repeated {
			if fp == "" || src.covered[start] {
				continue
			}
			members := windows[fp]
			groups = append(groups, d.region(members, d.extend(members)))
		}
	}
	return groups, duplicated
}

// prepare hashes every token once and splits each file into runs. Hashing a
// token per window instead would cost minTokens times as much.
func (d detector) prepare(files []*adapter.SourceFile) []*source {
	sorted := slices.Clone(files)
	slices.SortStableFunc(sorted, func(a, b *adapter.SourceFile) int {
		return strings.Compare(a.RelativePath, b.RelativePath)
	})
	sources := make([]*source, len(sorted))
	for index, file := range sorted {
		hashes := make([]uint64, len(file.Tokens))
		for position, token := range file.Tokens {
			hashes[position] = xxh3.HashString(token.Type)*rollingBase2 ^ xxh3.HashString(token.Value)
		}
		src := &source{
			file:      file,
			hashes:    hashes,
			runEnd:    make([]int, len(file.Tokens)),
			functions: functionSpans(file),
			repeated:  make([]string, len(file.Tokens)),
			covered:   make([]bool, len(file.Tokens)),
		}
		src.runs = tokenRuns(file)
		for _, r := range src.runs {
			for i := r.start; i < r.end; i++ {
				src.runEnd[i] = r.end
			}
		}
		sources[index] = src
	}
	return sources
}

// tokenRuns splits the tokens at import declarations, which are skipped.
func tokenRuns(file *adapter.SourceFile) []run {
	spans := slices.Clone(file.ImportSpans)
	slices.SortFunc(spans, func(a, b adapter.LineSpan) int { return a.Line - b.Line })
	var runs []run
	current, next := -1, 0
	for i, token := range file.Tokens {
		for next < len(spans) && spans[next].EndLine < token.Line {
			next++
		}
		inside := next < len(spans) && token.Line >= spans[next].Line && token.EndLine <= spans[next].EndLine
		switch {
		case inside && current >= 0:
			runs = append(runs, run{start: current, end: i})
			current = -1
		case !inside && current < 0:
			current = i
		}
	}
	if current >= 0 {
		runs = append(runs, run{start: current, end: len(file.Tokens)})
	}
	return runs
}

// functionSpans lists the file's functions from its measurements, so a copy can
// be named after the function that holds it.
func functionSpans(file *adapter.SourceFile) []functionSpan {
	var spans []functionSpan
	seen := map[string]bool{}
	for _, measurement := range file.Measurements {
		scope := measurement.Scope
		if scope.Kind != domain.ScopeFunction || seen[scope.Key] {
			continue
		}
		seen[scope.Key] = true
		spans = append(spans, functionSpan{symbol: scope.Symbol, line: scope.Line, endLine: scope.EndLine})
	}
	return spans
}

// enclosingSymbol names the innermost function containing a line, or "" for
// code outside every function.
func (s *source) enclosingSymbol(line int) string {
	symbol, size := "", -1
	for _, function := range s.functions {
		if line < function.line || line > function.endLine {
			continue
		}
		if extent := function.endLine - function.line; size < 0 || extent < size {
			symbol, size = function.symbol, extent
		}
	}
	return symbol
}

// repeatedWindows returns, per fingerprint, the windows that appear at least
// twice without overlapping each other in the same file.
func (d detector) repeatedWindows(sources []*source) map[string][]window {
	counts := make(map[windowKey]uint32)
	for _, src := range sources {
		d.eachWindow(src, func(_ int, key windowKey) {
			counts[key]++
		})
	}
	// Only keys seen twice can be clones, so only those get an exact fingerprint.
	candidates := make(map[string][]window)
	keys := make([]string, 0, d.minTokens)
	for _, src := range sources {
		tokens := src.file.Tokens
		d.eachWindow(src, func(start int, key windowKey) {
			if counts[key] < 2 {
				return
			}
			keys = keys[:0]
			for _, token := range tokens[start : start+d.minTokens] {
				keys = append(keys, token.Type+":"+token.Value)
			}
			fp := fingerprint.SHA256String(strings.Join(keys, "\x00"))
			candidates[fp] = append(candidates[fp], window{src: src, start: start})
		})
	}
	repeated := make(map[string][]window, len(candidates))
	for fp, windows := range candidates {
		if independent := d.independentWindows(windows); len(independent) >= 2 {
			repeated[fp] = independent
		}
	}
	return repeated
}

// eachWindow visits every eligible window of one file in ascending start
// order, rolling the hash forward in constant time per position.
func (d detector) eachWindow(src *source, visit func(start int, key windowKey)) {
	tokens := src.file.Tokens
	power1, power2 := uint64(1), uint64(1)
	for i := 0; i < d.minTokens-1; i++ {
		power1 *= rollingBase1
		power2 *= rollingBase2
	}
	for _, r := range src.runs {
		if r.end-r.start < d.minTokens {
			continue
		}
		var hash1, hash2 uint64
		for i := r.start; i < r.start+d.minTokens; i++ {
			hash1 = hash1*rollingBase1 + src.hashes[i]
			hash2 = hash2*rollingBase2 + src.hashes[i]
		}
		last := r.end - d.minTokens
		for start := r.start; ; start++ {
			if tokens[start+d.minTokens-1].EndLine-tokens[start].Line+1 >= d.minLines {
				visit(start, windowKey{hash1, hash2})
			}
			if start == last {
				break
			}
			hash1 = (hash1-src.hashes[start]*power1)*rollingBase1 + src.hashes[start+d.minTokens]
			hash2 = (hash2-src.hashes[start]*power2)*rollingBase2 + src.hashes[start+d.minTokens]
		}
	}
}

// independentWindows drops windows overlapping one already kept in the same
// file. Starts ascend per file, so only the last kept window can overlap.
func (d detector) independentWindows(candidates []window) []window {
	var independent []window
	lastEnd := make(map[*source]int, 4)
	for _, candidate := range candidates {
		if end, seen := lastEnd[candidate.src]; seen && candidate.start <= end {
			continue
		}
		independent = append(independent, candidate)
		lastEnd[candidate.src] = candidate.start + d.minTokens - 1
	}
	return independent
}

// extend grows the region shared by every member one token at a time, until
// a member reaches the end of its run or the members disagree.
func (d detector) extend(members []window) int {
	length := d.minTokens
	first := members[0]
	for {
		position := first.start + length
		if position >= first.src.runEnd[first.start] {
			return length
		}
		token := first.src.file.Tokens[position]
		for _, member := range members[1:] {
			other := member.start + length
			if other >= member.src.runEnd[member.start] {
				return length
			}
			candidate := member.src.file.Tokens[other]
			if candidate.Type != token.Type || candidate.Value != token.Value {
				return length
			}
		}
		length++
	}
}

// region records the members' copies of a length-token region and marks the
// tokens as covered, so a later window inside them does not seed another group.
func (d detector) region(members []window, length int) cloneGroup {
	group := cloneGroup{Tokens: length, Occurrences: make([]occurrence, 0, len(members))}
	keys := make([]string, 0, length)
	for i, member := range members {
		tokens := member.src.file.Tokens
		if i == 0 {
			for _, token := range tokens[member.start : member.start+length] {
				keys = append(keys, token.Type+":"+token.Value)
			}
			group.Fingerprint = fingerprint.SHA256String(strings.Join(keys, "\x00"))
		}
		for position := member.start; position < member.start+length; position++ {
			member.src.covered[position] = true
		}
		group.Occurrences = append(group.Occurrences, occurrence{
			File:       member.src.file.RelativePath,
			Symbol:     member.src.enclosingSymbol(tokens[member.start].Line),
			StartIndex: member.start,
			EndIndex:   member.start + length - 1,
			StartLine:  tokens[member.start].Line,
			EndLine:    tokens[member.start+length-1].EndLine,
		})
	}
	return group
}

func (w window) line() int {
	return w.src.file.Tokens[w.start].Line
}

func (w window) endLine(length int) int {
	return w.src.file.Tokens[w.start+length-1].EndLine
}
