package domain

import "slices"

// MetricPolicy is the project policy for one threshold metric. Thresholds are
// project decisions, not universal standards.
type MetricPolicy struct {
	Enabled   bool     `json:"enabled"`
	Threshold float64  `json:"threshold"`
	Severity  Severity `json:"severity"`
	Block     bool     `json:"block"`
}

// Exceeds reports whether a measured value breaches the policy.
func (p MetricPolicy) Exceeds(value float64) bool {
	return p.Enabled && value > p.Threshold
}

// DuplicationPolicy extends MetricPolicy with clone-window parameters.
type DuplicationPolicy struct {
	MetricPolicy
	MinTokens int `json:"minTokens"`
	MinLines  int `json:"minLines"`
}

// Boundary is a declared architecture layer with its permitted dependencies.
type Boundary struct {
	Name        string   `json:"name"`
	Include     []string `json:"include"`
	MayDependOn []string `json:"mayDependOn"`
}

// Allows reports whether the boundary may depend on the named target boundary.
func (b Boundary) Allows(target string) bool {
	return slices.Contains(b.MayDependOn, target)
}

// MetricsConfig groups all threshold policies.
type MetricsConfig struct {
	Cyclomatic         MetricPolicy      `json:"cyclomatic"`
	Cognitive          MetricPolicy      `json:"cognitive"`
	NestingDepth       MetricPolicy      `json:"nestingDepth"`
	FunctionLines      MetricPolicy      `json:"functionLines"`
	ParameterCount     MetricPolicy      `json:"parameterCount"`
	FileTokens         MetricPolicy      `json:"fileTokens"`
	DuplicationPercent DuplicationPolicy `json:"duplicationPercent"`
	FanOut             MetricPolicy      `json:"fanOut"`
}

// ArchitectureConfig declares boundaries and structural gate behaviour.
type ArchitectureConfig struct {
	Boundaries      []Boundary `json:"boundaries"`
	BlockCycles     bool       `json:"blockCycles"`
	BlockViolations bool       `json:"blockViolations"`
}

// CoverageConfig points at an optional LCOV tracefile.
type CoverageConfig struct {
	LCOVFile *string `json:"lcovFile"`
	Required bool    `json:"required"`
}

// SourceConfig selects source files with repository-relative glob patterns.
type SourceConfig struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

// Config is the validated repository configuration.
type Config struct {
	SchemaVersion string             `json:"schemaVersion"`
	Source        SourceConfig       `json:"source"`
	Metrics       MetricsConfig      `json:"metrics"`
	Architecture  ArchitectureConfig `json:"architecture"`
	Coverage      CoverageConfig     `json:"coverage"`
}

// PolicyFor returns the threshold policy that governs a metric, if any.
func (c Config) PolicyFor(id MetricID) (MetricPolicy, bool) {
	switch id {
	case MetricCyclomatic:
		return c.Metrics.Cyclomatic, true
	case MetricCognitive:
		return c.Metrics.Cognitive, true
	case MetricNestingDepth:
		return c.Metrics.NestingDepth, true
	case MetricFunctionLines:
		return c.Metrics.FunctionLines, true
	case MetricParameterCount:
		return c.Metrics.ParameterCount, true
	case MetricFileTokens:
		return c.Metrics.FileTokens, true
	case MetricTokenCloneDensity:
		return c.Metrics.DuplicationPercent.MetricPolicy, true
	case MetricModuleFanOut:
		return c.Metrics.FanOut, true
	default:
		return MetricPolicy{}, false
	}
}

// Clone returns a deep copy that can be mutated without affecting the original.
func (c Config) Clone() Config {
	cloned := c
	cloned.Source.Include = slices.Clone(c.Source.Include)
	cloned.Source.Exclude = slices.Clone(c.Source.Exclude)
	cloned.Architecture.Boundaries = make([]Boundary, len(c.Architecture.Boundaries))
	for i, boundary := range c.Architecture.Boundaries {
		cloned.Architecture.Boundaries[i] = Boundary{
			Name:        boundary.Name,
			Include:     slices.Clone(boundary.Include),
			MayDependOn: slices.Clone(boundary.MayDependOn),
		}
	}
	if c.Coverage.LCOVFile != nil {
		value := *c.Coverage.LCOVFile
		cloned.Coverage.LCOVFile = &value
	}
	return cloned
}
