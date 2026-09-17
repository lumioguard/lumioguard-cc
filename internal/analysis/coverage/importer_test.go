package coverage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/analysis"
	"github.com/lumioguard/lumioguard-cc/internal/config"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

type fixture struct {
	root   string
	source string
	file   *adapter.SourceFile
	input  analysis.Input
}

func newFixture(t *testing.T, lcov string) fixture {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "sample.ts")
	if err := os.WriteFile(source, []byte("export const value = (ready: boolean) => ready ? 1 : 0;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-10 * time.Second)
	if err := os.Chtimes(source, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lcov.info"), []byte(lcov), 0o644); err != nil {
		t.Fatal(err)
	}
	file := &adapter.SourceFile{AbsolutePath: source, RelativePath: "sample.ts", Code: "x"}
	cfg := config.Default()
	lcovFile := "lcov.info"
	cfg.Coverage.LCOVFile = &lcovFile
	return fixture{root: root, source: source, file: file, input: analysis.Input{Root: root, Files: []*adapter.SourceFile{file}, Config: cfg}}
}

func find(result analysis.Result, id domain.MetricID) *domain.Measurement {
	for i := range result.Measurements {
		if result.Measurements[i].MetricID == id {
			return &result.Measurements[i]
		}
	}
	return nil
}

func TestImportKeepsDenominatorsAndCalculatesChangedCoverage(t *testing.T) {
	f := newFixture(t, "TN:\nSF:sample.ts\nDA:1,1\nDA:2,0\nBRDA:1,0,0,1\nBRDA:1,0,1,0\nend_of_record\n")
	f.input.ChangedLines = analysis.ChangedLines{"sample.ts": analysis.LineSet{1: {}}}
	result, err := New().Analyze(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
	line := find(result, domain.MetricCoverageLine)
	if line == nil || *line.Value != 50 || line.Status != domain.StatusMeasured || line.Evidence["covered"] != 1 || line.Evidence["total"] != 2 {
		t.Fatalf("unexpected line coverage %+v", line)
	}
	if branch := find(result, domain.MetricCoverageBranch); branch == nil || *branch.Value != 50 {
		t.Fatalf("unexpected branch coverage %+v", branch)
	}
	if changedLine := find(result, domain.MetricCoverageChangedLine); changedLine == nil || *changedLine.Value != 100 {
		t.Fatalf("unexpected changed line coverage %+v", changedLine)
	}
	if changedBranch := find(result, domain.MetricCoverageChangedBranch); changedBranch == nil || *changedBranch.Value != 50 {
		t.Fatalf("unexpected changed branch coverage %+v", changedBranch)
	}
}

func TestEmptyBranchDenominatorIsNotApplicable(t *testing.T) {
	f := newFixture(t, "SF:sample.ts\nDA:1,1\nend_of_record\n")
	result, err := New().Analyze(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	branch := find(result, domain.MetricCoverageBranch)
	if branch == nil || branch.Value != nil || branch.Status != domain.StatusNotApplicable {
		t.Fatalf("expected not_applicable branch coverage, got %+v", branch)
	}
}

func TestStaleReportIsNotUsed(t *testing.T) {
	f := newFixture(t, "SF:sample.ts\nDA:1,1\nend_of_record\n")
	future := time.Now().Add(10 * time.Second)
	if err := os.Chtimes(f.source, future, future); err != nil {
		t.Fatal(err)
	}
	result, err := New().Analyze(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Measurements) != 0 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "coverage.report_stale" {
		t.Fatalf("expected stale diagnostic only, got %+v / %+v", result.Measurements, result.Diagnostics)
	}
}

func TestRequiredMissingReportIsIncomplete(t *testing.T) {
	f := newFixture(t, "SF:sample.ts\nDA:1,1\nend_of_record\n")
	missing := "missing.info"
	f.input.Config.Coverage.LCOVFile = &missing
	f.input.Config.Coverage.Required = true
	result, err := New().Analyze(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "coverage.report_unavailable" || !result.Diagnostics[0].IsRequiredError() {
		t.Fatalf("expected required unavailable diagnostic, got %+v", result.Diagnostics)
	}
}

func TestUnmappedReportIsDisclosed(t *testing.T) {
	f := newFixture(t, "SF:other.ts\nDA:1,1\nend_of_record\n")
	result, err := New().Analyze(context.Background(), f.input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "coverage.no_sources_mapped" || result.Diagnostics[0].Severity != domain.SeverityWarning {
		t.Fatalf("expected no_sources_mapped warning, got %+v", result.Diagnostics)
	}
}

func TestParseLCOVHandlesUnknownBranchOutcomes(t *testing.T) {
	records, err := ParseLCOV(strings.NewReader("SF:a.ts\nDA:1,3\nDA:2,0\nBRDA:1,0,0,-\nBRDA:1,0,1,2\nend_of_record\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || len(records[0].Lines) != 2 || len(records[0].Branches) != 2 || countKnownBranches(records[0]) != 1 {
		t.Fatalf("unexpected records %+v", records)
	}
}

func countKnownBranches(record FileRecord) int {
	total := 0
	for range record.KnownBranches() {
		total++
	}
	return total
}
