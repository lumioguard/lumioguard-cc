package engine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/golang"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/java"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/python"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/typescript"
	"github.com/lumioguard/lumioguard-cc/internal/analysis"
	"github.com/lumioguard/lumioguard-cc/internal/analysis/coverage"
	"github.com/lumioguard/lumioguard-cc/internal/analysis/duplication"
	"github.com/lumioguard/lumioguard-cc/internal/analysis/graph"
	"github.com/lumioguard/lumioguard-cc/internal/analysis/tokens"
	"github.com/lumioguard/lumioguard-cc/internal/baseline"
	"github.com/lumioguard/lumioguard-cc/internal/comparison"
	"github.com/lumioguard/lumioguard-cc/internal/config"
	"github.com/lumioguard/lumioguard-cc/internal/discovery"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/engine"
	"github.com/lumioguard/lumioguard-cc/internal/git"
	"github.com/lumioguard/lumioguard-cc/internal/policy"
)

type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

type fixedIDs struct{}

func (fixedIDs) NewRunID() string { return "run-1" }

func newAnalyzer(opts ...engine.Option) *engine.Analyzer {
	return engine.New(engine.Dependencies{
		Tool:       domain.ToolInfo{Name: "test", Version: "0"},
		Adapters:   adapter.NewRegistry(typescript.New(), python.New(), java.New(), golang.New()),
		Discoverer: discovery.NewWalker(),
		Repository: []analysis.RepositoryAnalyzer{graph.New("recheck"), duplication.New("recheck"), tokens.New(), coverage.New()},
		Git:        git.NewClient(),
		Comparer:   comparison.New(),
		Policy:     policy.New("recheck"),
	}, opts...)
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseFailureIsIncomplete(t *testing.T) {
	root := t.TempDir()
	write(t, root, "invalid.ts", "export function broken( {")
	report, err := newAnalyzer().Analyze(context.Background(), engine.Request{Root: root, Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	if report.Policy.Status != domain.PolicyIncomplete {
		t.Fatalf("expected incomplete, got %+v", report.Policy)
	}
	found := false
	for _, d := range report.Diagnostics {
		if d.Code == "typescript.parse_failed" && d.Required {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected required parse diagnostic, got %+v", report.Diagnostics)
	}
	if report.Adapters[0].Complete {
		t.Fatalf("adapter must be reported as incomplete: %+v", report.Adapters)
	}
}

func TestUnsupportedLanguageIsNeverGuessed(t *testing.T) {
	root := t.TempDir()
	write(t, root, "sample.rb", "def value\n  1\nend\n")
	cfg := config.Default()
	cfg.Source.Include = []string{"**/*.rb"}
	report, err := newAnalyzer().Analyze(context.Background(), engine.Request{Root: root, Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if report.Scope.FilesDiscovered != 1 || report.Scope.FilesAnalyzed != 0 || report.Policy.Status != domain.PolicyIncomplete {
		t.Fatalf("unexpected scope/policy %+v %+v", report.Scope, report.Policy)
	}
	if report.Diagnostics[0].Code != "adapter.unsupported_language" || report.Diagnostics[0].File != "sample.rb" {
		t.Fatalf("unexpected diagnostic %+v", report.Diagnostics[0])
	}
}

func TestMixedLanguageRepositoryIsAnalyzedInOneInvocation(t *testing.T) {
	root := t.TempDir()
	write(t, root, "web/app.ts", "import { helper } from './helper';\nexport function run(x: number) { return x > 0 ? helper() : 0; }\n")
	write(t, root, "web/helper.ts", "export function helper() { return 1; }\n")
	write(t, root, "svc/pkg/__init__.py", "")
	write(t, root, "svc/pkg/core.py", "from . import util\n\ndef work(items):\n    return [util.f(i) for i in items if i]\n")
	write(t, root, "svc/pkg/util.py", "def f(i):\n    return i\n")
	write(t, root, "api/src/main/java/com/acme/App.java", "package com.acme;\n\npublic class App {\n    public int go(boolean a) { Util.help(); return a ? 1 : 0; }\n}\n")
	write(t, root, "api/src/main/java/com/acme/Util.java", "package com.acme;\n\nclass Util { static void help() {} }\n")
	write(t, root, "go.mod", "module example.com/mixed\n\ngo 1.27\n")
	write(t, root, "cmd/run.go", "package main\n\nimport \"example.com/mixed/lib\"\n\nfunc Run(x int) int {\n\tif x > 0 {\n\t\treturn lib.One()\n\t}\n\treturn 0\n}\n")
	write(t, root, "lib/lib.go", "package lib\n\nfunc One() int { return 1 }\n")
	report, err := newAnalyzer().Analyze(context.Background(), engine.Request{Root: root, Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	if report.Scope.FilesDiscovered != 9 || report.Scope.FilesAnalyzed != 9 {
		t.Fatalf("unexpected scope %+v", report.Scope)
	}
	if report.Scope.Languages["TypeScript"] != 2 || report.Scope.Languages["Python"] != 3 || report.Scope.Languages["Java"] != 2 || report.Scope.Languages["Go"] != 2 {
		t.Fatalf("unexpected language counts %+v", report.Scope.Languages)
	}
	if len(report.Adapters) != 4 {
		t.Fatalf("expected four adapters, got %+v", report.Adapters)
	}
	for _, summary := range report.Adapters {
		if !summary.Complete {
			t.Fatalf("adapter %s incomplete: %+v", summary.ID, report.Diagnostics)
		}
	}
	if report.Policy.Status != domain.PolicyPassed {
		t.Fatalf("expected passed, got %+v %+v", report.Policy, report.Diagnostics)
	}
	fanOut := map[string]float64{}
	symbols := map[string]bool{}
	for _, m := range report.Measurements {
		if m.MetricID == domain.MetricModuleFanOut {
			fanOut[m.Scope.File] = *m.Value
		}
		if m.MetricID == domain.MetricCyclomatic {
			symbols[m.Scope.Key] = true
		}
	}
	for file, want := range map[string]float64{"web/app.ts": 1, "svc/pkg/core.py": 1, "api/src/main/java/com/acme/App.java": 1, "cmd/run.go": 1} {
		if fanOut[file] != want {
			t.Errorf("fan-out %s = %v, want %v", file, fanOut[file], want)
		}
	}
	for _, key := range []string{"web/app.ts::run", "svc/pkg/core.py::work", "api/src/main/java/com/acme/App.java::App.go", "cmd/run.go::Run"} {
		if !symbols[key] {
			t.Errorf("missing function scope %s", key)
		}
	}
}

func TestBaselineComparisonClassifiesWorsenedDebt(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Metrics.Cyclomatic.Threshold = 1
	write(t, root, "sample.ts", "export function check(value: number) { if (value > 0) return 1; return 0; }")
	analyzer := newAnalyzer()
	original, err := analyzer.Analyze(context.Background(), engine.Request{Root: root, Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	store := baseline.NewStore("0")
	write(t, root, "sample.ts", "export function check(value: number) { if (value > 0) return 1; if (value < 0) return -1; return 0; }")
	current, err := analyzer.Analyze(context.Background(), engine.Request{Root: root, Config: cfg, Baseline: store.FromReport("initial", original), BaselineName: "initial"})
	if err != nil {
		t.Fatal(err)
	}
	var metric *domain.Measurement
	for i := range current.Measurements {
		if current.Measurements[i].MetricID == domain.MetricCyclomatic && current.Measurements[i].Scope.Symbol == "check" {
			metric = &current.Measurements[i]
		}
	}
	if metric == nil || *metric.Baseline != 2 || *metric.Value != 3 || *metric.Delta != 1 || metric.Classification != domain.ClassificationWorsened {
		t.Fatalf("unexpected measurement %+v", metric)
	}
	var finding *domain.Finding
	for i := range current.Findings {
		if current.Findings[i].RuleID == domain.RuleID(domain.MetricCyclomatic) {
			finding = &current.Findings[i]
		}
	}
	if finding == nil || *finding.Baseline != 2 || *finding.Current != 3 || finding.Classification != domain.ClassificationWorsened {
		t.Fatalf("unexpected finding %+v", finding)
	}
	if current.Comparison.Mode != domain.ComparisonBaseline || !current.Comparison.Comparable || *current.Comparison.Reference != "initial" {
		t.Fatalf("unexpected comparison %+v", current.Comparison)
	}
}

func TestRepeatedAnalysisIsDeterministicApartFromRunMetadata(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.ts", "import { b } from './b';\nexport function a(x: number) { return x > 1 ? b() : 0; }\n")
	write(t, root, "b.ts", "export function b() { return 1; }\n")
	write(t, root, "c.py", "import d\n\ndef c(x):\n    return d.d() if x else 0\n")
	write(t, root, "d.py", "def d():\n    return 1\n")
	write(t, root, "E.java", "public class E { int e(int x) { return x > 1 ? 1 : 0; } }\n")
	analyzer := newAnalyzer(engine.WithClock(fixedClock{at: time.Unix(0, 0)}), engine.WithIDGenerator(fixedIDs{}), engine.WithWorkers(1))
	first, err := analyzer.Analyze(context.Background(), engine.Request{Root: root, Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	parallel := newAnalyzer(engine.WithClock(fixedClock{at: time.Unix(0, 0)}), engine.WithIDGenerator(fixedIDs{}), engine.WithWorkers(8))
	second, err := parallel.Analyze(context.Background(), engine.Request{Root: root, Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	one, _ := json.Marshal(first)
	two, _ := json.Marshal(second)
	if string(one) != string(two) {
		t.Fatalf("reports differ:\n%s\n%s", one, two)
	}
	if len(first.Measurements) == 0 || first.Policy.Status != domain.PolicyPassed {
		t.Fatalf("unexpected report %+v %+v", first.Policy, first.Diagnostics)
	}
	if first.Snapshot.FileHashes["a.ts"] == "" || first.Snapshot.ConfigHash == "" || first.Snapshot.ID == "" {
		t.Fatalf("snapshot hashes missing: %+v", first.Snapshot)
	}
}
