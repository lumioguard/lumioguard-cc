package tokens

import (
	"context"
	"testing"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/analysis"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
)

func tokens(n int) []adapter.Token {
	out := make([]adapter.Token, n)
	for i := range out {
		out[i] = adapter.Token{Type: "IDENT", Value: "x", Line: 1, EndLine: 1, Index: i}
	}
	return out
}

func find(result analysis.Result, id domain.MetricID, key string) *domain.Measurement {
	for i := range result.Measurements {
		if result.Measurements[i].MetricID == id && result.Measurements[i].Scope.Key == key {
			return &result.Measurements[i]
		}
	}
	return nil
}

func TestCountsEveryFileAndTheTotal(t *testing.T) {
	result, err := New().Analyze(context.Background(), analysis.Input{Files: []*adapter.SourceFile{
		{RelativePath: "a.go", Tokens: tokens(3)},
		{RelativePath: "b.go", Tokens: tokens(5)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := find(result, domain.MetricFileTokens, "b.go"); got == nil || *got.Value != 5 || got.Unit != domain.UnitCount {
		t.Fatalf("unexpected file measurement %+v", got)
	}
	total := find(result, domain.MetricTotalTokens, "repository")
	if total == nil || *total.Value != 8 || total.Evidence["files"] != 2 {
		t.Fatalf("unexpected total %+v", total)
	}
}

// A file that failed to parse has no tokens; reporting 0 would reward broken code.
func TestUnanalyzedFileHasNoNumberAndSpoilsTheTotal(t *testing.T) {
	result, err := New().Analyze(context.Background(), analysis.Input{Files: []*adapter.SourceFile{
		{RelativePath: "ok.go", Tokens: tokens(3)},
		{RelativePath: "broken.go", Diagnostics: []domain.Diagnostic{domain.RequiredError("go.parse_failed", "broken.go", "bad")}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if broken := find(result, domain.MetricFileTokens, "broken.go"); broken == nil || broken.Value != nil || broken.Status != domain.StatusUnavailable {
		t.Fatalf("unexpected measurement for the broken file %+v", broken)
	}
	if total := find(result, domain.MetricTotalTokens, "repository"); total == nil || total.Value != nil || total.Status != domain.StatusUnavailable {
		t.Fatalf("total must be unavailable, got %+v", total)
	}
}

func TestNoFilesIsNotApplicable(t *testing.T) {
	result, err := New().Analyze(context.Background(), analysis.Input{})
	if err != nil {
		t.Fatal(err)
	}
	if total := find(result, domain.MetricTotalTokens, "repository"); total == nil || total.Status != domain.StatusNotApplicable {
		t.Fatalf("unexpected total %+v", total)
	}
}
