package adapter

import "github.com/lumiostack/lumioguard-cc/internal/domain"

// MetricVariants names the exact algorithm variants an adapter used.
type MetricVariants struct {
	Cyclomatic     string
	Cognitive      string
	NestingDepth   string
	FunctionLines  string
	ParameterCount string
}

// FunctionMetrics is the per-function result every adapter produces.
type FunctionMetrics struct {
	Scope                    domain.Scope
	Cyclomatic               int
	Decisions                any
	Cognitive                int
	Increments               any
	NestingDepth             int
	SourceLines              int
	ParameterCount           int
	Variants                 MetricVariants
	CognitiveAnalyzer        string
	CognitiveAnalyzerVersion string
}

// FunctionMeasurements renders the five per-function measurements in the
// shared shape so reports look identical across languages.
func FunctionMeasurements(m FunctionMetrics) []domain.Measurement {
	return []domain.Measurement{
		{
			MetricID: domain.MetricCyclomatic,
			Variant:  m.Variants.Cyclomatic,
			Category: domain.CategoryComplexity,
			Scope:    m.Scope,
			Value:    domain.Float(float64(m.Cyclomatic)),
			Unit:     domain.UnitCount,
			Status:   domain.StatusMeasured,
			Evidence: domain.Evidence{"decisions": m.Decisions},
		},
		{
			MetricID: domain.MetricCognitive,
			Variant:  m.Variants.Cognitive,
			Category: domain.CategoryComplexity,
			Scope:    m.Scope,
			Value:    domain.Float(float64(m.Cognitive)),
			Unit:     domain.UnitCount,
			Status:   domain.StatusMeasured,
			Evidence: domain.Evidence{
				"analyzer":        m.CognitiveAnalyzer,
				"analyzerVersion": m.CognitiveAnalyzerVersion,
				"increments":      m.Increments,
			},
		},
		{
			MetricID: domain.MetricNestingDepth,
			Variant:  m.Variants.NestingDepth,
			Category: domain.CategoryComplexity,
			Scope:    m.Scope,
			Value:    domain.Float(float64(m.NestingDepth)),
			Unit:     domain.UnitCount,
			Status:   domain.StatusMeasured,
		},
		{
			MetricID: domain.MetricFunctionLines,
			Variant:  m.Variants.FunctionLines,
			Category: domain.CategorySize,
			Scope:    m.Scope,
			Value:    domain.Float(float64(m.SourceLines)),
			Unit:     domain.UnitLines,
			Status:   domain.StatusMeasured,
		},
		{
			MetricID: domain.MetricParameterCount,
			Variant:  m.Variants.ParameterCount,
			Category: domain.CategorySize,
			Scope:    m.Scope,
			Value:    domain.Float(float64(m.ParameterCount)),
			Unit:     domain.UnitCount,
			Status:   domain.StatusMeasured,
		},
	}
}
