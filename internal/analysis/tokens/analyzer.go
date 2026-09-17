// Package tokens measures how much source there is to read: the lexical tokens
// of every analyzed file and their total, so a comparison shows what a change
// added to the amount of code a person or an agent has to take in.
package tokens

import (
	"context"
	"fmt"

	"github.com/lumioguard/lumioguard-cc/internal/analysis"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

const (
	// Name identifies the analyzer in error messages.
	Name = "source-tokens"
	// Variant is the exact counting rule: the adapter's clone tokens, which never include comments.
	Variant = "adapter-lexical-tokens-without-comments-v1"
)

// Analyzer implements analysis.RepositoryAnalyzer for source token counts.
type Analyzer struct{}

// New creates the analyzer.
func New() *Analyzer {
	return &Analyzer{}
}

// Name implements analysis.RepositoryAnalyzer.
func (a *Analyzer) Name() string {
	return Name
}

// Analyze implements analysis.RepositoryAnalyzer. A file that could not be
// analyzed gets no number, and then neither does the total.
func (a *Analyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	measurements := make([]domain.Measurement, 0, len(in.Files)+1)
	total, unavailable := 0, 0
	for _, file := range in.Files {
		measurement := domain.Measurement{
			MetricID: domain.MetricFileTokens,
			Variant:  Variant,
			Category: domain.CategorySize,
			Scope:    domain.FileScope(file.RelativePath),
			Unit:     domain.UnitCount,
			Status:   domain.StatusMeasured,
		}
		if file.HasRequiredDiagnostic() {
			measurement.Status = domain.StatusUnavailable
			measurement.Reason = "The file could not be analyzed"
			unavailable++
		} else {
			measurement.Value = domain.Float(float64(len(file.Tokens)))
			total += len(file.Tokens)
		}
		measurements = append(measurements, measurement)
	}

	repository := domain.Measurement{
		MetricID: domain.MetricTotalTokens,
		Variant:  Variant,
		Category: domain.CategorySize,
		Scope:    domain.RepositoryScope(),
		Unit:     domain.UnitCount,
		Status:   domain.StatusMeasured,
		Evidence: domain.Evidence{"files": len(in.Files)},
	}
	switch {
	case len(in.Files) == 0:
		repository.Status = domain.StatusNotApplicable
		repository.Reason = "No source files were analyzed"
	case unavailable > 0:
		repository.Status = domain.StatusUnavailable
		repository.Reason = fmt.Sprintf("%d files could not be analyzed", unavailable)
	default:
		repository.Value = domain.Float(float64(total))
	}
	return analysis.Result{Measurements: append(measurements, repository)}, nil
}
