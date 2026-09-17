// Package worklist turns a report's active findings into an ordered list of
// places to fix. It never changes the findings; it groups them by place and
// orders the places, so a person or an agent can start at the top.
package worklist

import (
	"cmp"
	"encoding/json"
	"slices"
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/glob"
)

// Options select and cap the entries.
type Options struct {
	// Rules keeps findings of these rules only; empty keeps every rule.
	Rules []domain.RuleID
	// Paths keeps findings whose files match one of these globs; empty keeps every file.
	Paths []string
	// Top caps each section; 0 keeps every entry.
	Top int
}

// Item is one finding inside an entry.
type Item struct {
	RuleID         domain.RuleID         `json:"ruleId"`
	Current        *float64              `json:"current"`
	Threshold      *float64              `json:"threshold"`
	Unit           domain.Unit           `json:"unit"`
	Blocking       bool                  `json:"blocking"`
	Classification domain.Classification `json:"classification"`
	Message        string                `json:"message"`
}

// Copy is one location of a duplicated block.
type Copy struct {
	File      string `json:"file"`
	Symbol    string `json:"symbol,omitempty"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
}

// Entry is one place to fix: a function, a file, a cycle, a boundary violation
// or a duplicated block. Nested entries are functions inside a function entry;
// their findings are already counted in the outer function's numbers.
type Entry struct {
	Scope    domain.Scope `json:"scope"`
	Findings []Item       `json:"findings"`
	Nested   []Entry      `json:"nested,omitempty"`
	// Rules counts the distinct rules broken here and in nested entries.
	Rules int `json:"rules"`
	// Excess is the largest current/threshold ratio here and in nested entries; 0 when no threshold applies.
	Excess float64 `json:"excess"`
	Copies []Copy  `json:"copies,omitempty"`
}

// Worklist is the ordered result.
type Worklist struct {
	Comparison domain.Comparison   `json:"comparison"`
	Policy     domain.PolicyResult `json:"policy"`
	// Findings counts the active findings that passed the filters.
	Findings int `json:"findings"`
	// Structure lists dependency cycles and boundary violations: fix these first.
	Structure []Entry `json:"structure"`
	// Hotspots lists functions and files, the ones breaking the most rules first.
	Hotspots []Entry `json:"hotspots"`
	// Clones lists duplicated blocks, the ones taking the most lines first.
	Clones []Entry `json:"clones"`
	// Density is the project's clone density finding, when it exceeds its threshold.
	Density *Item `json:"density,omitempty"`
	// Omitted counts the entries cut by Options.Top.
	Omitted int `json:"omitted"`
}

// Build orders the report's active findings. Hotspots come first by the number
// of rules they break and then by how far they exceed their own limit; that
// ordering is a heuristic for where to start, not a debt score.
func Build(report *domain.Report, options Options) Worklist {
	list := Worklist{Comparison: report.Comparison, Policy: report.Policy}
	var structure, clones, hotspots []domain.Finding
	for _, finding := range report.ActiveFindings() {
		if !options.keeps(finding) {
			continue
		}
		list.Findings++
		switch finding.RuleID {
		case domain.RuleDependencyCycle, domain.RuleBoundaryViolation:
			structure = append(structure, finding)
		case domain.RuleTokenClone:
			clones = append(clones, finding)
		case domain.RuleID(domain.MetricTokenCloneDensity):
			density := item(finding)
			list.Density = &density
		default:
			hotspots = append(hotspots, finding)
		}
	}
	list.Structure = structureEntries(structure)
	list.Hotspots = hotspotEntries(hotspots)
	list.Clones = cloneEntries(clones)
	if options.Top > 0 {
		list.Structure, list.Omitted = limit(list.Structure, options.Top, list.Omitted)
		list.Hotspots, list.Omitted = limit(list.Hotspots, options.Top, list.Omitted)
		list.Clones, list.Omitted = limit(list.Clones, options.Top, list.Omitted)
	}
	return list
}

func (o Options) keeps(finding domain.Finding) bool {
	if len(o.Rules) > 0 && !slices.Contains(o.Rules, finding.RuleID) {
		return false
	}
	if len(o.Paths) == 0 {
		return true
	}
	for _, file := range files(finding) {
		if glob.MatchesAny(file, o.Paths) {
			return true
		}
	}
	return false
}

// evidenceLocations is the part of a finding's evidence that names files: the
// copies of a duplicated block, or the members of a cycle.
type evidenceLocations struct {
	Occurrences []Copy   `json:"occurrences"`
	Modules     []string `json:"modules"`
}

func locations(finding domain.Finding) evidenceLocations {
	var decoded evidenceLocations
	encoded, err := json.Marshal(finding.Evidence)
	if err != nil {
		return decoded
	}
	_ = json.Unmarshal(encoded, &decoded)
	return decoded
}

func files(finding domain.Finding) []string {
	if finding.Scope.File != "" {
		return []string{finding.Scope.File}
	}
	found := locations(finding)
	result := slices.Clone(found.Modules)
	for _, copy := range found.Occurrences {
		result = append(result, copy.File)
	}
	return result
}

func item(finding domain.Finding) Item {
	return Item{
		RuleID:         finding.RuleID,
		Current:        finding.Current,
		Threshold:      finding.Threshold,
		Unit:           finding.Unit,
		Blocking:       finding.Blocking,
		Classification: finding.Classification,
		Message:        finding.Message,
	}
}

func structureEntries(findings []domain.Finding) []Entry {
	entries := make([]Entry, 0, len(findings))
	for _, finding := range findings {
		entries = append(entries, Entry{Scope: finding.Scope, Findings: []Item{item(finding)}, Rules: 1})
	}
	slices.SortStableFunc(entries, func(a, b Entry) int {
		return cmp.Or(
			strings.Compare(string(a.Findings[0].RuleID), string(b.Findings[0].RuleID)),
			strings.Compare(a.Scope.Key, b.Scope.Key),
		)
	})
	return entries
}

func cloneEntries(findings []domain.Finding) []Entry {
	entries := make([]Entry, 0, len(findings))
	for _, finding := range findings {
		entry := Entry{Scope: finding.Scope, Findings: []Item{item(finding)}, Rules: 1, Copies: locations(finding).Occurrences}
		entries = append(entries, entry)
	}
	slices.SortStableFunc(entries, func(a, b Entry) int {
		return cmp.Or(
			cmp.Compare(value(b.Findings[0]), value(a.Findings[0])),
			strings.Compare(a.Scope.Key, b.Scope.Key),
		)
	})
	return entries
}

// node is an entry while its nested functions are still being attached.
type node struct {
	entry    Entry
	children []*node
}

// hotspotEntries groups function and file findings by place, folds functions
// into the function that contains them, and orders the result.
func hotspotEntries(findings []domain.Finding) []Entry {
	byKey := map[string]*node{}
	var order []string
	for _, finding := range findings {
		key := finding.Scope.Key
		current, seen := byKey[key]
		if !seen {
			current = &node{entry: Entry{Scope: finding.Scope}}
			byKey[key] = current
			order = append(order, key)
		}
		current.entry.Findings = append(current.entry.Findings, item(finding))
	}
	var roots []*node
	for _, key := range order {
		if parent := enclosing(byKey[key].entry.Scope, byKey); parent != nil {
			parent.children = append(parent.children, byKey[key])
			continue
		}
		roots = append(roots, byKey[key])
	}
	entries := make([]Entry, 0, len(roots))
	for _, root := range roots {
		entries = append(entries, materialise(root))
	}
	slices.SortStableFunc(entries, func(a, b Entry) int {
		return cmp.Or(
			cmp.Compare(b.Rules, a.Rules),
			cmp.Compare(b.Excess, a.Excess),
			strings.Compare(a.Scope.File, b.Scope.File),
			cmp.Compare(a.Scope.Line, b.Scope.Line),
			strings.Compare(a.Scope.Key, b.Scope.Key),
		)
	})
	return entries
}

// enclosing finds the longest function containing scope in the same file that
// has findings of its own, by the "outer.inner" symbol convention every
// adapter uses.
func enclosing(scope domain.Scope, byKey map[string]*node) *node {
	if scope.Kind != domain.ScopeFunction {
		return nil
	}
	symbol := scope.Symbol
	for {
		dot := strings.LastIndex(symbol, ".")
		if dot < 0 {
			return nil
		}
		symbol = symbol[:dot]
		if parent, ok := byKey[domain.FunctionScope(scope.File, symbol, 0, 0).Key]; ok {
			return parent
		}
	}
}

// materialise turns a node into an entry with its nested entries, and fills
// Rules and Excess from the entry's own findings and the nested ones.
func materialise(current *node) Entry {
	entry := current.entry
	slices.SortStableFunc(entry.Findings, func(a, b Item) int { return strings.Compare(string(a.RuleID), string(b.RuleID)) })
	rules := map[domain.RuleID]bool{}
	for _, finding := range entry.Findings {
		rules[finding.RuleID] = true
		entry.Excess = max(entry.Excess, excess(finding))
	}
	for _, child := range current.children {
		nested := materialise(child)
		for _, finding := range nested.Findings {
			rules[finding.RuleID] = true
		}
		entry.Excess = max(entry.Excess, nested.Excess)
		entry.Nested = append(entry.Nested, nested)
	}
	slices.SortStableFunc(entry.Nested, func(a, b Entry) int { return cmp.Compare(a.Scope.Line, b.Scope.Line) })
	entry.Rules = len(rules)
	return entry
}

func excess(finding Item) float64 {
	if finding.Current == nil || finding.Threshold == nil || *finding.Threshold <= 0 {
		return 0
	}
	return *finding.Current / *finding.Threshold
}

func value(finding Item) float64 {
	if finding.Current == nil {
		return 0
	}
	return *finding.Current
}

func limit(entries []Entry, top, omitted int) ([]Entry, int) {
	if len(entries) <= top {
		return entries, omitted
	}
	return entries[:top], omitted + len(entries) - top
}
