package domain

// MetricID is the permanent identifier of a metric in the catalog.
type MetricID string

// Metric identifiers. They are stable across releases; a changed algorithm is
// expressed through the measurement variant, never through a new ID.
const (
	MetricCyclomatic            MetricID = "complexity.cyclomatic"
	MetricCognitive             MetricID = "complexity.cognitive"
	MetricNestingDepth          MetricID = "complexity.nesting_depth"
	MetricFunctionLines         MetricID = "size.function_lines"
	MetricParameterCount        MetricID = "size.parameter_count"
	MetricFileTokens            MetricID = "size.file_tokens"
	MetricTotalTokens           MetricID = "size.total_tokens"
	MetricTokenCloneDensity     MetricID = "duplication.token_clone_density"
	MetricModuleFanIn           MetricID = "coupling.module_fan_in"
	MetricModuleFanOut          MetricID = "coupling.module_fan_out"
	MetricCoverageLine          MetricID = "coverage.line"
	MetricCoverageBranch        MetricID = "coverage.branch"
	MetricCoverageChangedLine   MetricID = "coverage.changed_line"
	MetricCoverageChangedBranch MetricID = "coverage.changed_branch"
)

// RuleID identifies a rule that produces findings. Threshold rules reuse the
// metric identifier they evaluate.
type RuleID string

// Rule identifiers for structural rules that are not simple thresholds.
const (
	RuleDependencyCycle   RuleID = "dependency.cycle"
	RuleBoundaryViolation RuleID = "architecture.boundary_violation"
	RuleTokenClone        RuleID = "duplication.token_clone"
)
