// Package app implements the use cases behind each CLI command. Services depend
// on small interfaces, wired with real implementations in the composition root.
package app

import (
	"context"

	"github.com/lumiostack/lumioguard-cc/internal/config"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/engine"
)

// ConfigLoader reads and initialises repository configuration.
type ConfigLoader interface {
	Load(root string) (config.Loaded, error)
	WriteDefault(root string) (string, error)
}

// Engine runs an analysis.
type Engine interface {
	Analyze(ctx context.Context, req engine.Request) (*domain.Report, error)
}

// BaselineRepository persists and loads named baselines.
type BaselineRepository interface {
	Load(root, name string) (*domain.Baseline, error)
	Save(root, name string, report *domain.Report, replace bool) (string, error)
	FromReport(name string, report *domain.Report) *domain.Baseline
}

// GitReferences resolves and exports Git commits.
type GitReferences interface {
	Version(ctx context.Context) (string, error)
	RevParse(ctx context.Context, root, ref string) (string, error)
	ExportTree(ctx context.Context, root, commit, destination string) error
}

// Comparer classifies a report against a baseline.
type Comparer interface {
	Compare(report *domain.Report, baseline *domain.Baseline, mode domain.ComparisonMode, reference string) *domain.Report
}

// PolicyEvaluator recomputes the gate after a comparison.
type PolicyEvaluator interface {
	Result(findings []domain.Finding, incomplete bool) domain.PolicyResult
}
