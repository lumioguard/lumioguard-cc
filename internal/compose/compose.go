// Package compose is the composition root: it wires concrete implementations
// into the application. It is the only place that knows every package.
package compose

import (
	"github.com/lumioguard/lumioguard-cc/internal/analysis"
	"github.com/lumioguard/lumioguard-cc/internal/analysis/coverage"
	"github.com/lumioguard/lumioguard-cc/internal/analysis/duplication"
	"github.com/lumioguard/lumioguard-cc/internal/analysis/graph"
	"github.com/lumioguard/lumioguard-cc/internal/analysis/tokens"
	"github.com/lumioguard/lumioguard-cc/internal/app"
	"github.com/lumioguard/lumioguard-cc/internal/baseline"
	"github.com/lumioguard/lumioguard-cc/internal/comparison"
	"github.com/lumioguard/lumioguard-cc/internal/config"
	"github.com/lumioguard/lumioguard-cc/internal/discovery"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/engine"
	"github.com/lumioguard/lumioguard-cc/internal/explain"
	"github.com/lumioguard/lumioguard-cc/internal/git"
	"github.com/lumioguard/lumioguard-cc/internal/policy"
	"github.com/lumioguard/lumioguard-cc/internal/product"
)

// NewApplication builds the production application graph.
func NewApplication() *app.Application {
	tool := domain.ToolInfo{Name: product.Name, Version: product.Version}
	registry := NewRegistry()
	walker := discovery.NewWalker()
	gitClient := git.NewClient()
	comparer := comparison.New()
	evaluator := policy.New(product.RecheckCommand)
	analyzer := engine.New(engine.Dependencies{
		Tool:       tool,
		Adapters:   registry,
		Discoverer: walker,
		Repository: []analysis.RepositoryAnalyzer{
			graph.New(product.RecheckCommand),
			duplication.New(product.RecheckCommand),
			tokens.New(),
			coverage.New(),
		},
		Git:      gitClient,
		Comparer: comparer,
		Policy:   evaluator,
	})
	return app.New(app.Dependencies{
		Tool:       tool,
		Config:     config.NewLoader(),
		Engine:     analyzer,
		Baselines:  baseline.NewStore(product.Version),
		Git:        gitClient,
		Comparer:   comparer,
		Policy:     evaluator,
		Adapters:   registry,
		Discoverer: walker,
		Catalog:    explain.NewCatalog(),
	})
}
