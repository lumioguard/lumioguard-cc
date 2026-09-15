package duplication

import (
	"context"
	"strings"
	"testing"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/typescript"
	"github.com/lumiostack/lumioguard-cc/internal/analysis"
	"github.com/lumiostack/lumioguard-cc/internal/config"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
)

const repeated = `export function calculate(value: number) {
  const adjusted = value + 10;
  return adjusted * 2;
}`

func TestExactTokenClonesProduceDensityAndFindings(t *testing.T) {
	cfg := config.Default()
	cfg.Metrics.DuplicationPercent.MinTokens = 8
	cfg.Metrics.DuplicationPercent.MinLines = 2
	files := []*adapter.SourceFile{
		typescript.Analyze("a.ts", "/r/a.ts", repeated),
		typescript.Analyze("b.ts", "/r/b.ts", strings.Replace(repeated, "calculate", "calculateAgain", 1)),
	}
	result, err := New("recheck").Analyze(context.Background(), analysis.Input{Files: files, Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	density := result.Measurements[0]
	if density.MetricID != domain.MetricTokenCloneDensity || density.Value == nil || *density.Value <= 0 {
		t.Fatalf("expected positive density, got %+v", density)
	}
	if density.Evidence["cloneGroups"].(int) < 1 {
		t.Fatalf("expected clone groups in evidence, got %+v", density.Evidence)
	}
	if len(result.Findings) == 0 || result.Findings[0].RuleID != domain.RuleTokenClone {
		t.Fatalf("expected clone findings, got %+v", result.Findings)
	}
}

func TestNoSourcesIsNotApplicable(t *testing.T) {
	result, err := New("recheck").Analyze(context.Background(), analysis.Input{Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Measurements[0].Status != domain.StatusNotApplicable || result.Measurements[0].Value != nil {
		t.Fatalf("expected not_applicable, got %+v", result.Measurements[0])
	}
}

func TestFindingsRequireThresholdBreach(t *testing.T) {
	cfg := config.Default()
	cfg.Metrics.DuplicationPercent.MinTokens = 8
	cfg.Metrics.DuplicationPercent.MinLines = 2
	cfg.Metrics.DuplicationPercent.Threshold = 100
	files := []*adapter.SourceFile{
		typescript.Analyze("a.ts", "/r/a.ts", repeated),
		typescript.Analyze("b.ts", "/r/b.ts", repeated),
	}
	result, err := New("recheck").Analyze(context.Background(), analysis.Input{Files: files, Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("no findings expected below threshold, got %+v", result.Findings)
	}
}
