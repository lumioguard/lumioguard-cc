package graph

import (
	"context"
	"path"
	"strings"
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/analysis"
	"github.com/lumioguard/lumioguard-cc/internal/config"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

// fakeResolver resolves "./x" to a ".ts" file beside the importer, treats bare
// names as external and missing relative paths as unresolved.
type fakeResolver struct{}

func (fakeResolver) Resolve(source *adapter.SourceFile, imp adapter.Import, index *adapter.ModuleIndex) adapter.Resolution {
	if !strings.HasPrefix(imp.Specifier, ".") {
		return adapter.External()
	}
	candidate := path.Join(path.Dir(source.RelativePath), imp.Specifier) + ".ts"
	if index.Has(candidate) {
		return adapter.Internal(candidate)
	}
	return adapter.NotResolved()
}

type fakeAdapter struct{}

func (fakeAdapter) ID() string                       { return "fake" }
func (fakeAdapter) Version() string                  { return "1" }
func (fakeAdapter) Languages() []string              { return []string{"Fake"} }
func (fakeAdapter) Analyzers() map[string]string     { return map[string]string{} }
func (fakeAdapter) Supports(string) bool             { return true }
func (fakeAdapter) Resolver() adapter.ImportResolver { return fakeResolver{} }
func (fakeAdapter) AnalyzeFile(context.Context, string, string) (*adapter.SourceFile, error) {
	return nil, nil
}

func file(relative string, imports ...string) *adapter.SourceFile {
	f := &adapter.SourceFile{RelativePath: relative, Code: "x"}
	for i, specifier := range imports {
		f.Imports = append(f.Imports, adapter.Import{Specifier: specifier, Line: i + 1, Kind: adapter.ImportStatic})
	}
	return f
}

func input(cfg domain.Config, files ...*adapter.SourceFile) analysis.Input {
	return analysis.Input{Files: files, Config: cfg, Adapters: adapter.NewRegistry(fakeAdapter{})}
}

func measurementValue(result analysis.Result, id domain.MetricID, relative string) float64 {
	for _, m := range result.Measurements {
		if m.MetricID == id && m.Scope.File == relative {
			return *m.Value
		}
	}
	return -1
}

func TestFanInFanOutAndCycles(t *testing.T) {
	result, err := New("recheck").Analyze(context.Background(), input(config.Default(), file("a.ts", "./b"), file("b.ts", "./a"), file("c.ts", "./a", "lodash")))
	if err != nil {
		t.Fatal(err)
	}
	if got := measurementValue(result, domain.MetricModuleFanOut, "a.ts"); got != 1 {
		t.Fatalf("fan-out a.ts = %v", got)
	}
	if got := measurementValue(result, domain.MetricModuleFanIn, "a.ts"); got != 2 {
		t.Fatalf("fan-in a.ts = %v", got)
	}
	var cycles []domain.Finding
	for _, f := range result.Findings {
		if f.RuleID == domain.RuleDependencyCycle {
			cycles = append(cycles, f)
		}
	}
	if len(cycles) != 1 || cycles[0].Scope.Key != "a.ts|b.ts" || !cycles[0].Blocking {
		t.Fatalf("expected one blocking a|b cycle, got %+v", cycles)
	}
	infos := 0
	for _, d := range result.Diagnostics {
		if d.Code == "dependency.external_not_analyzed" {
			infos++
		}
	}
	if infos != 1 {
		t.Fatalf("external imports must be disclosed once, got %+v", result.Diagnostics)
	}
}

func TestBoundaryViolationsUseFirstMatchingBoundary(t *testing.T) {
	cfg := config.Default()
	cfg.Architecture.Boundaries = []domain.Boundary{
		{Name: "domain", Include: []string{"domain/**"}, MayDependOn: []string{"domain"}},
		{Name: "infra", Include: []string{"infra/**"}, MayDependOn: []string{"domain", "infra"}},
	}
	result, err := New("recheck").Analyze(context.Background(), input(cfg, file("domain/order.ts", "../infra/db"), file("infra/db.ts", "../domain/order")))
	if err != nil {
		t.Fatal(err)
	}
	var violations []domain.Finding
	for _, f := range result.Findings {
		if f.RuleID == domain.RuleBoundaryViolation {
			violations = append(violations, f)
		}
	}
	if len(violations) != 1 || violations[0].Scope.Key != "domain/order.ts->infra/db.ts" || !violations[0].Blocking || violations[0].Scope.Line != 1 {
		t.Fatalf("expected one blocking domain->infra violation, got %+v", violations)
	}
}

func TestUnresolvedImportsAreWarnings(t *testing.T) {
	result, err := New("recheck").Analyze(context.Background(), input(config.Default(), file("a.ts", "./missing")))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "dependency.import_unresolved" || result.Diagnostics[0].Required {
		t.Fatalf("expected an advisory unresolved diagnostic, got %+v", result.Diagnostics)
	}
}

func TestFilesWithoutAdapterAreSkipped(t *testing.T) {
	result, err := New("recheck").Analyze(context.Background(), analysis.Input{Files: []*adapter.SourceFile{file("a.ts", "./b")}, Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 0 || measurementValue(result, domain.MetricModuleFanOut, "a.ts") != 0 {
		t.Fatalf("expected no edges without resolvers, got %+v", result)
	}
}
