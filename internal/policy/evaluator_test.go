package policy

import (
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/config"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

func TestThresholdFindingsRespectPolicy(t *testing.T) {
	cfg := config.Default()
	cfg.Metrics.Cyclomatic.Threshold = 2
	cfg.Metrics.Cyclomatic.Block = true
	measurements := []domain.Measurement{
		{MetricID: domain.MetricCyclomatic, Variant: "v1", Scope: domain.FunctionScope("a.ts", "f", 1, 2), Value: domain.Float(3), Unit: domain.UnitCount, Status: domain.StatusMeasured},
		{MetricID: domain.MetricCyclomatic, Variant: "v1", Scope: domain.FunctionScope("a.ts", "g", 3, 4), Value: domain.Float(2), Unit: domain.UnitCount, Status: domain.StatusMeasured},
		{MetricID: domain.MetricCognitive, Variant: "v1", Scope: domain.FunctionScope("a.ts", "f", 1, 2), Value: nil, Unit: domain.UnitCount, Status: domain.StatusUnavailable},
		{MetricID: domain.MetricModuleFanIn, Variant: "v1", Scope: domain.FileScope("a.ts"), Value: domain.Float(99), Unit: domain.UnitCount, Status: domain.StatusMeasured},
	}
	findings := New("recheck").ThresholdFindings(measurements, cfg)
	if len(findings) != 1 {
		t.Fatalf("expected exactly one finding, got %+v", findings)
	}
	f := findings[0]
	if f.RuleID != domain.RuleID(domain.MetricCyclomatic) || !f.Blocking || *f.Current != 3 || *f.Threshold != 2 || f.Recheck != "recheck" {
		t.Fatalf("unexpected finding %+v", f)
	}
	if f.Message != "complexity.cyclomatic is 3 count; the configured threshold is 2" {
		t.Fatalf("unexpected message %q", f.Message)
	}
	if f.Evidence["metricVariant"] != "v1" {
		t.Fatalf("metric variant must be recorded as evidence: %+v", f.Evidence)
	}
}

func TestResultCountsOnlyNewOrWorsenedBlockingFindings(t *testing.T) {
	evaluator := New("recheck")
	findings := []domain.Finding{
		{Blocking: true, Classification: domain.ClassificationExisting},
		{Blocking: true, Classification: domain.ClassificationResolved},
		{Blocking: false, Classification: domain.ClassificationNew},
		{Blocking: false, Classification: domain.ClassificationResolved},
	}
	result := evaluator.Result(findings, false)
	if result.Status != domain.PolicyPassed || result.BlockingFindings != 0 || result.WarningFindings != 1 {
		t.Fatalf("unexpected result %+v", result)
	}
	findings = append(findings, domain.Finding{Blocking: true, Classification: domain.ClassificationWorsened})
	if result := evaluator.Result(findings, false); result.Status != domain.PolicyFailed || result.BlockingFindings != 1 {
		t.Fatalf("worsened blocking finding must fail: %+v", result)
	}
	if result := evaluator.Result(findings, true); result.Status != domain.PolicyIncomplete {
		t.Fatalf("incomplete must take precedence: %+v", result)
	}
}
