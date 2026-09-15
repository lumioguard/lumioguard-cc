// Package compose is the composition root: it wires concrete implementations
// into the application. It is the only place that knows every package.
package compose

import (
	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/golang"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/java"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/python"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/typescript"
	"github.com/lumiostack/lumioguard-cc/internal/analysis"
	"github.com/lumiostack/lumioguard-cc/internal/analysis/coverage"
	"github.com/lumiostack/lumioguard-cc/internal/analysis/duplication"
	"github.com/lumiostack/lumioguard-cc/internal/analysis/graph"
	"github.com/lumiostack/lumioguard-cc/internal/analysis/tokens"
	"github.com/lumiostack/lumioguard-cc/internal/app"
	"github.com/lumiostack/lumioguard-cc/internal/baseline"
	"github.com/lumiostack/lumioguard-cc/internal/comparison"
	"github.com/lumiostack/lumioguard-cc/internal/config"
	"github.com/lumiostack/lumioguard-cc/internal/discovery"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/engine"
	"github.com/lumiostack/lumioguard-cc/internal/explain"
	"github.com/lumiostack/lumioguard-cc/internal/git"
	"github.com/lumiostack/lumioguard-cc/internal/policy"
	"github.com/lumiostack/lumioguard-cc/internal/product"
)

// NewRegistry lists the installed language adapters in priority order.
func NewRegistry() *adapter.Registry {
	return adapter.NewRegistry(typescript.New(), python.New(), java.New(), golang.New())
}

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
