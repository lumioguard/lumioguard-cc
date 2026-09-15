package duplication

import (
	"maps"
	"slices"
	"strings"

	"github.com/zeebo/xxh3"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/analysis"
	"github.com/lumiostack/lumioguard-cc/internal/fingerprint"
)

// occurrence is one location of a clone window. It is serialised as evidence.
type occurrence struct {
	File       string `json:"file"`
	StartIndex int    `json:"startIndex"`
	EndIndex   int    `json:"endIndex"`
	StartLine  int    `json:"startLine"`
	EndLine    int    `json:"endLine"`
}

// cloneGroup is an accepted set of independent occurrences of one window.
type cloneGroup struct {
	Fingerprint string
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

// detector finds exact repeated token windows.
type detector struct {
	minTokens int
	minLines  int
}

func newDetector(minTokens, minLines int) detector {
	return detector{minTokens: minTokens, minLines: minLines}
}

// detect returns clone groups in fingerprint order and the duplicated lines per
// file. A group needs two non-overlapping occurrences and one new line.
func (d detector) detect(files []*adapter.SourceFile) ([]cloneGroup, map[string]analysis.LineSet) {
	hashes := d.tokenHashes(files)
	windows := d.collectWindows(files, hashes, d.countWindows(files, hashes))

	fingerprints := slices.Sorted(maps.Keys(windows))

	duplicated := make(map[string]analysis.LineSet, len(files))
	for _, file := range files {
		duplicated[file.RelativePath] = analysis.LineSet{}
	}
	var accepted []cloneGroup
	for _, fp := range fingerprints {
		independent := independentOccurrences(windows[fp])
		if len(independent) < 2 || !introducesNewLines(independent, duplicated) {
			continue
		}
		accepted = append(accepted, cloneGroup{Fingerprint: fp, Occurrences: independent})
		for _, item := range independent {
			lines := duplicated[item.File]
			for line := item.StartLine; line <= item.EndLine; line++ {
				lines.Add(line)
			}
		}
	}
	return accepted, duplicated
}

// tokenHashes hashes every token once. Hashing a token per window instead
// would cost minTokens times as much, which dominated the whole analysis.
func (d detector) tokenHashes(files []*adapter.SourceFile) [][]uint64 {
	hashes := make([][]uint64, len(files))
	for index, file := range files {
		perFile := make([]uint64, len(file.Tokens))
		for position, token := range file.Tokens {
			perFile[position] = xxh3.HashString(token.Type)*rollingBase2 ^ xxh3.HashString(token.Value)
		}
		hashes[index] = perFile
	}
	return hashes
}

// countWindows counts each rolling hash. Only keys seen twice can be clones, so
// only those get an exact fingerprint.
func (d detector) countWindows(files []*adapter.SourceFile, hashes [][]uint64) map[windowKey]uint32 {
	counts := make(map[windowKey]uint32)
	for index, file := range files {
		d.eachWindow(file, hashes[index], func(_ int, key windowKey) {
			counts[key]++
		})
	}
	return counts
}

func (d detector) collectWindows(files []*adapter.SourceFile, hashes [][]uint64, counts map[windowKey]uint32) map[string][]occurrence {
	windows := make(map[string][]occurrence)
	keys := make([]string, 0, d.minTokens)
	for index, file := range files {
		tokens := file.Tokens
		d.eachWindow(file, hashes[index], func(start int, key windowKey) {
			if counts[key] < 2 {
				return
			}
			keys = keys[:0]
			for _, token := range tokens[start : start+d.minTokens] {
				keys = append(keys, token.Type+":"+token.Value)
			}
			fp := fingerprint.SHA256String(strings.Join(keys, "\x00"))
			windows[fp] = append(windows[fp], occurrence{
				File:       file.RelativePath,
				StartIndex: start,
				EndIndex:   start + d.minTokens - 1,
				StartLine:  tokens[start].Line,
				EndLine:    tokens[start+d.minTokens-1].EndLine,
			})
		})
	}
	return windows
}

// eachWindow visits every eligible window of one file in ascending start
// order, rolling the hash forward in constant time per position.
func (d detector) eachWindow(file *adapter.SourceFile, hashes []uint64, visit func(start int, key windowKey)) {
	tokens := file.Tokens
	if len(tokens) < d.minTokens {
		return
	}
	power1, power2 := uint64(1), uint64(1)
	for i := 0; i < d.minTokens-1; i++ {
		power1 *= rollingBase1
		power2 *= rollingBase2
	}
	var hash1, hash2 uint64
	for i := 0; i < d.minTokens; i++ {
		hash1 = hash1*rollingBase1 + hashes[i]
		hash2 = hash2*rollingBase2 + hashes[i]
	}
	last := len(tokens) - d.minTokens
	for start := 0; ; start++ {
		if tokens[start+d.minTokens-1].EndLine-tokens[start].Line+1 >= d.minLines {
			visit(start, windowKey{hash1, hash2})
		}
		if start == last {
			return
		}
		hash1 = (hash1-hashes[start]*power1)*rollingBase1 + hashes[start+d.minTokens]
		hash2 = (hash2-hashes[start]*power2)*rollingBase2 + hashes[start+d.minTokens]
	}
}

// independentOccurrences drops windows overlapping one already kept in the same
// file. Starts ascend per file, so only the last kept window can overlap.
func independentOccurrences(candidates []occurrence) []occurrence {
	var independent []occurrence
	lastEnd := make(map[string]int, 4)
	for _, candidate := range candidates {
		if end, seen := lastEnd[candidate.File]; seen && candidate.StartIndex <= end {
			continue
		}
		independent = append(independent, candidate)
		lastEnd[candidate.File] = candidate.EndIndex
	}
	return independent
}

func introducesNewLines(occurrences []occurrence, duplicated map[string]analysis.LineSet) bool {
	for _, item := range occurrences {
		lines := duplicated[item.File]
		for line := item.StartLine; line <= item.EndLine; line++ {
			if !lines.Has(line) {
				return true
			}
		}
	}
	return false
}
