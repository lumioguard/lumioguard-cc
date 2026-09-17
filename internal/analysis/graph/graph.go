// Package graph builds the module dependency graph from each adapter's resolver
// and evaluates coupling, cycles and architecture boundaries.
package graph

import (
	"fmt"
	"maps"
	"slices"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/analysis"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

type edgeKey struct {
	source string
	target string
}

// moduleGraph is the resolved internal dependency graph of one analysis.
type moduleGraph struct {
	modules     []string
	edges       map[string]map[string]struct{}
	sorted      map[string][]string
	edgeLines   map[edgeKey]int
	diagnostics []domain.Diagnostic
}

// build resolves every import with its file's adapter. External imports are
// counted; internal-looking imports that do not resolve become warnings.
func build(in analysis.Input) *moduleGraph {
	index := adapter.NewModuleIndex(in.Files)
	g := &moduleGraph{
		edges:     make(map[string]map[string]struct{}, len(in.Files)),
		edgeLines: make(map[edgeKey]int),
	}
	for _, file := range in.Files {
		g.edges[file.RelativePath] = map[string]struct{}{}
	}
	g.modules = slices.Sorted(maps.Keys(g.edges))

	external := 0
	for _, file := range in.Files {
		resolver := resolverFor(in.Adapters, file)
		if resolver == nil {
			continue
		}
		for _, imported := range file.Imports {
			resolution := resolver.Resolve(file, imported, index)
			switch resolution.Kind {
			case adapter.ResolvedInternal:
				g.edges[file.RelativePath][resolution.Target] = struct{}{}
				key := edgeKey{source: file.RelativePath, target: resolution.Target}
				if _, seen := g.edgeLines[key]; !seen {
					g.edgeLines[key] = imported.Line
				}
			case adapter.ResolvedExternal:
				external++
			default:
				g.diagnostics = append(g.diagnostics, domain.Advisory(
					"dependency.import_unresolved",
					file.RelativePath,
					fmt.Sprintf("Could not resolve import '%s' at line %d", imported.Specifier, imported.Line),
					domain.SeverityWarning,
				))
			}
		}
	}
	g.sorted = make(map[string][]string, len(g.edges))
	for module, targets := range g.edges {
		g.sorted[module] = slices.Sorted(maps.Keys(targets))
	}

	if external > 0 {
		g.diagnostics = append(g.diagnostics, domain.Advisory(
			"dependency.external_not_analyzed",
			"",
			fmt.Sprintf("%d imports were treated as external (packages, standard libraries or unresolvable aliases); see .documentations/rules/dependencies.md#how-imports-are-found for how imports are resolved in each language", external),
			domain.SeverityInfo,
		))
	}
	return g
}

func resolverFor(finder adapter.Finder, file *adapter.SourceFile) adapter.ImportResolver {
	if finder == nil {
		return nil
	}
	languageAdapter, ok := finder.Find(file.RelativePath)
	if !ok {
		return nil
	}
	return languageAdapter.Resolver()
}

// targets returns the sorted dependencies of a module. The order is fixed once
// in build because every module is asked for its targets more than once.
func (g *moduleGraph) targets(module string) []string {
	return g.sorted[module]
}

// fanIn counts distinct dependants per module.
func (g *moduleGraph) fanIn() map[string]int {
	counts := make(map[string]int, len(g.modules))
	for _, module := range g.modules {
		counts[module] = 0
	}
	for _, targets := range g.edges {
		for target := range targets {
			counts[target]++
		}
	}
	return counts
}
