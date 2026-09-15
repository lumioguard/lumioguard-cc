// Package explain is the metric and rule catalog: definitions, variants,
// limitations and sources shown by `lumioguard-cc explain`.
package explain

import "github.com/lumiostack/lumioguard-cc/internal/domain"

// Classification records how established a definition is.
type Classification string

// Documented classifications of definitions.
const (
	Classical       Classification = "classical"
	VendorDefined   Classification = "vendor-defined"
	Conventional    Classification = "conventional"
	ProjectSpecific Classification = "project-specific"
)

// Entry explains one metric or rule.
type Entry struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Classification Classification `json:"classification"`
	Definition     string         `json:"definition"`
	Variant        string         `json:"variant"`
	Limitations    string         `json:"limitations"`
	Source         string         `json:"source"`
}

// Catalog is an ordered, read-only set of entries.
type Catalog struct {
	entries []Entry
	byID    map[string]Entry
}

// NewCatalog returns the catalog for this release.
func NewCatalog() *Catalog {
	catalog := &Catalog{byID: map[string]Entry{}}
	for _, entry := range entries {
		catalog.entries = append(catalog.entries, entry)
		catalog.byID[entry.ID] = entry
	}
	return catalog
}

// Lookup finds an entry by metric or rule identifier.
func (c *Catalog) Lookup(id string) (Entry, bool) {
	entry, ok := c.byID[id]
	return entry, ok
}

// IDs lists every identifier in catalog order.
func (c *Catalog) IDs() []string {
	ids := make([]string, 0, len(c.entries))
	for _, entry := range c.entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

var entries = []Entry{
	{
		ID:             string(domain.MetricCyclomatic),
		Name:           "Cyclomatic complexity",
		Classification: Classical,
		Definition:     "One plus the number of decision points inside a function.",
		Variant:        "Counts if (including else-if and elif), loops, catch/except handlers, non-default case labels, conditional expressions and binary logical operators (&&, ||, ?? in JavaScript/TypeScript; and, or plus comprehension clauses in Python; &&, || in Java). Nested functions reset scope. Same algorithm for every language; see .documentations/rules/complexity.md#language-notes for the construct mapping.",
		Limitations:    "Syntactic paths do not establish behavioral risk or correctness. Logical-operator counting varies between tools.",
		Source:         "McCabe (1976), A Complexity Measure, IEEE Transactions on Software Engineering.",
	},
	{
		ID:             string(domain.MetricCognitive),
		Name:           "Cognitive complexity",
		Classification: VendorDefined,
		Definition:     "Sonar's measure of how difficult control flow is to understand: structural increments plus nesting penalties.",
		Variant:        "Independent Go implementation of the published Cognitive Complexity specification (SonarSource, v1.5) shared by JavaScript/TypeScript, Python and Java: increments for if/else if/else, ternaries, switch and match, loops (and Python loop else), catch/except, labelled jumps and sequences of && and || (and, or), with nesting increments; nested functions and lambdas raise the nesting level and their increments count toward the enclosing function; each nested function is also measured on its own. Recursion, ?? and comprehension clauses are not counted.",
		Limitations:    "This is not SonarJS output. Values were validated against hand-calculated fixtures from the specification, not against SonarQube. Results can change with variant updates.",
		Source:         "https://www.sonarsource.com/resources/cognitive-complexity/",
	},
	{
		ID:             string(domain.MetricNestingDepth),
		Name:           "Maximum nesting depth",
		Classification: Conventional,
		Definition:     "The deepest nesting of control-flow constructs inside a function.",
		Variant:        "Counts if, loops, switch/match, catch/except handlers and conditional expressions; else-if and elif are structurally nested; comprehension clauses and nested functions do not count. Same algorithm for every language.",
		Limitations:    "Tools differ on which constructs contribute to depth.",
		Source:         "Project metric definition; see .documentations/rules/complexity.md#nesting-depth.",
	},
	{
		ID:             string(domain.MetricFunctionLines),
		Name:           "Function source lines",
		Classification: Conventional,
		Definition:     "Nonblank, noncomment physical source lines intersecting a function.",
		Variant:        "From the first token of the declaration (including modifiers and annotations in TypeScript and Java; from def in Python, decorators excluded) to its end; multiline statements count each nonblank line; nested function text remains in the enclosing function size.",
		Limitations:    "Source lines are a size signal, not a direct quality judgment.",
		Source:         "Project metric definition; see .documentations/rules/size.md#function-length.",
	},
	{
		ID:             string(domain.MetricParameterCount),
		Name:           "Parameter count",
		Classification: Conventional,
		Definition:     "Number of declared formal parameter slots.",
		Variant:        "Destructured, default, rest, *args, **kwargs, keyword-only and TypeScript `this` parameters each count as one slot; Python self and cls count; Java receiver parameters do not.",
		Limitations:    "Some APIs legitimately require many parameters; object parameters can hide input complexity.",
		Source:         "Project metric definition; see .documentations/rules/size.md#parameter-count.",
	},
	{
		ID:             string(domain.MetricFileTokens),
		Name:           "File tokens",
		Classification: ProjectSpecific,
		Definition:     "Number of lexical tokens in a file, without comments: how much code there is to read in one file.",
		Variant:        "The same token stream the duplication rule uses: identifiers, keywords, literals and punctuation, with comments and whitespace removed. A statement-ending newline in Go and Python counts as one token.",
		Limitations:    "Lexical tokens are a proxy for what a language model reads, not a tokenizer's exact count. Comments are not counted.",
		Source:         "Project metric definition; see .documentations/rules/size.md#file-tokens.",
	},
	{
		ID:             string(domain.MetricTotalTokens),
		Name:           "Total tokens",
		Classification: ProjectSpecific,
		Definition:     "Sum of size.file_tokens over every analyzed file; compared with an earlier version, its delta is how much code a change added or removed.",
		Variant:        "See size.file_tokens. Unavailable when any file could not be analyzed, so an incomplete run never looks smaller.",
		Limitations:    "Reported only; it has no threshold.",
		Source:         "Project metric definition; see .documentations/rules/size.md#file-tokens.",
	},
	{
		ID:             string(domain.MetricTokenCloneDensity),
		Name:           "Token clone density",
		Classification: Conventional,
		Definition:     "Unique lines participating in eligible exact token clones divided by nonblank source lines.",
		Variant:        "Exact token values with configurable minimum token and line spans; comments are never tokens; regular expressions, template literal parts and JSX text are single opaque tokens.",
		Limitations:    "Exact clones miss renamed clones; repeated code does not always warrant abstraction.",
		Source:         "Project variant inspired by token-based duplication measures; see .documentations/rules/duplication.md.",
	},
	{
		ID:             string(domain.MetricModuleFanIn),
		Name:           "Module fan-in",
		Classification: Classical,
		Definition:     "Count of distinct analyzed modules with resolved dependencies on a module.",
		Variant:        "Imports resolved by each language adapter: relative JavaScript/TypeScript imports; Python package imports (relative, absolute and src layouts); Java declared, wildcard and same-package type references.",
		Limitations:    "Runtime, package, alias and library dependencies are not represented.",
		Source:         "Henry and Kafura information-flow family; exact project variant in .documentations/rules/dependencies.md#fan-out-and-fan-in.",
	},
	{
		ID:             string(domain.MetricModuleFanOut),
		Name:           "Module fan-out",
		Classification: Classical,
		Definition:     "Count of distinct analyzed modules a module depends on.",
		Variant:        "Imports resolved by each language adapter; see .documentations/rules/dependencies.md#how-imports-are-found for the per-language resolution rules.",
		Limitations:    "A high value is a review signal, not proof of poor design.",
		Source:         "Henry and Kafura information-flow family; exact project variant in .documentations/rules/dependencies.md#fan-out-and-fan-in.",
	},
	{
		ID:             string(domain.MetricCoverageLine),
		Name:           "Line coverage",
		Classification: Conventional,
		Definition:     "Eligible executable lines with at least one hit divided by eligible executable lines.",
		Variant:        "Imported from LCOV DA records.",
		Limitations:    "Execution does not prove assertion quality or correctness.",
		Source:         "LCOV tracefile semantics; exact import rules in .documentations/rules/coverage.md.",
	},
	{
		ID:             string(domain.MetricCoverageBranch),
		Name:           "Branch coverage",
		Classification: Conventional,
		Definition:     "Known branch outcomes with at least one hit divided by known branch outcomes.",
		Variant:        "Imported from LCOV BRDA records; unknown '-' outcomes are excluded.",
		Limitations:    "Instrumentation defines eligible branches and may differ by tool.",
		Source:         "LCOV tracefile semantics; exact import rules in .documentations/rules/coverage.md.",
	},
	{
		ID:             string(domain.MetricCoverageChangedLine),
		Name:           "Changed-line coverage",
		Classification: ProjectSpecific,
		Definition:     "Line coverage restricted to lines added since the Git comparison base.",
		Variant:        "LCOV DA records intersected with Git-added lines and untracked files; available with --base only.",
		Limitations:    "Named baselines store no source text, so changed lines cannot be reconstructed from them.",
		Source:         "Project scoping of conventional line coverage; see .documentations/rules/coverage.md.",
	},
	{
		ID:             string(domain.MetricCoverageChangedBranch),
		Name:           "Changed-branch coverage",
		Classification: ProjectSpecific,
		Definition:     "Branch coverage restricted to branch locations on lines added since the Git comparison base.",
		Variant:        "LCOV BRDA records intersected with Git-added lines; available with --base only.",
		Limitations:    "Branch locations are instrumenter-defined.",
		Source:         "Project scoping of conventional branch coverage; see .documentations/rules/coverage.md.",
	},
	{
		ID:             string(domain.RuleDependencyCycle),
		Name:           "Dependency cycle",
		Classification: Classical,
		Definition:     "A strongly connected component of two or more resolved internal modules.",
		Variant:        "Tarjan's algorithm over the imports each language adapter resolves; self-imports alone are not reported.",
		Limitations:    "A clean graph does not prove the runtime architecture is acyclic.",
		Source:         "Established directed-graph property; see .documentations/rules/dependencies.md#dependency-cycles.",
	},
	{
		ID:             string(domain.RuleBoundaryViolation),
		Name:           "Architecture boundary violation",
		Classification: ProjectSpecific,
		Definition:     "A resolved dependency from a declared boundary to a boundary it may not depend on.",
		Variant:        "First matching include pattern owns a file; the target boundary must appear in mayDependOn.",
		Limitations:    "Files outside every boundary and external imports are not enforced.",
		Source:         "Project architecture rule declared in the configuration.",
	},
	{
		ID:             string(domain.RuleTokenClone),
		Name:           "Duplicated token block",
		Classification: Conventional,
		Definition:     "One accepted group of exact token-window clones, reported when clone density exceeds the project threshold.",
		Variant:        "See duplication.token_clone_density.",
		Limitations:    "Evidence for review, not a mandate to abstract.",
		Source:         "Project variant; see .documentations/rules/duplication.md.",
	},
}
