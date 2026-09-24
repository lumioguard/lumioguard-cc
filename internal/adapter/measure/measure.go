// Package measure turns a function of the shared structure model into the
// five per-function measurements, so adapters do not repeat the assembly.
package measure

import (
	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/sourcetext"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

// Profile names the algorithm variants and the cognitive analyzer an adapter reports.
type Profile struct {
	Variants                 adapter.MetricVariants
	CognitiveAnalyzer        string
	CognitiveAnalyzerVersion string
}

// Function measures one function. Comments must be sorted by Start; they are
// excluded from the function's source lines.
func Function(code string, record structure.Record, comments []sourcetext.Range, profile Profile) []domain.Measurement {
	function := record.Function
	cyclomatic := structure.Cyclomatic(function)
	cognitive := structure.Cognitive(function)
	return adapter.FunctionMeasurements(adapter.FunctionMetrics{
		Scope:                    record.Scope,
		Cyclomatic:               cyclomatic.Value,
		Decisions:                cyclomatic.Decisions,
		Cognitive:                cognitive.Value,
		Increments:               cognitive.Increments,
		NestingDepth:             structure.NestingDepth(function),
		SourceLines:              sourcetext.CountSourceLines(code, function.Start, function.End, comments),
		ParameterCount:           function.Parameters,
		Variants:                 profile.Variants,
		CognitiveAnalyzer:        profile.CognitiveAnalyzer,
		CognitiveAnalyzerVersion: profile.CognitiveAnalyzerVersion,
	})
}
