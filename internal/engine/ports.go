// Package engine runs one analysis from discovery to ordered, compared results.
// It depends on interfaces only; the composition root supplies implementations.
package engine

import (
	"context"
	"time"

	"github.com/lumiostack/lumioguard-cc/internal/analysis"
	"github.com/lumiostack/lumioguard-cc/internal/discovery"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/fingerprint"
)

// Clock supplies the current time; tests substitute a fixed clock.
type Clock interface {
	Now() time.Time
}

// IDGenerator supplies run identifiers.
type IDGenerator interface {
	NewRunID() string
}

// FileDiscoverer selects source files under a root.
type FileDiscoverer interface {
	Discover(root string, source domain.SourceConfig) (discovery.Result, error)
}

// GitInspector reads Git state without modifying it.
type GitInspector interface {
	Inspect(ctx context.Context, root, base string) (domain.GitSnapshot, []domain.Diagnostic)
	ChangedLines(ctx context.Context, root, base string) (analysis.ChangedLines, string, []domain.Diagnostic)
}

// Comparer classifies a report against a baseline.
type Comparer interface {
	Compare(report *domain.Report, baseline *domain.Baseline, mode domain.ComparisonMode, reference string) *domain.Report
}

// PolicyEvaluator applies thresholds and computes the gate result.
type PolicyEvaluator interface {
	ThresholdFindings(measurements []domain.Measurement, cfg domain.Config) []domain.Finding
	Result(findings []domain.Finding, incomplete bool) domain.PolicyResult
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

type randomIDs struct{}

func (randomIDs) NewRunID() string {
	return fingerprint.NewRunID()
}
