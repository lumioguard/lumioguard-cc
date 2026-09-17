package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

func TestFeedbackSaysHowManyFindingsItLeftOut(t *testing.T) {
	report := &domain.Report{Policy: domain.PolicyResult{Status: domain.PolicyFailed}}
	for i := range 7 {
		report.Findings = append(report.Findings, domain.Finding{
			RuleID:         domain.RuleID(domain.MetricCyclomatic),
			Blocking:       true,
			Classification: domain.ClassificationNew,
			Scope:          domain.FunctionScope("a.ts", fmt.Sprintf("f%d", i), i+1, i+2),
			Message:        "too complex",
		})
	}
	message := feedback(report, CheckRequest{GitBase: "HEAD"}, "C:/tools/lumioguard-cc.exe")
	if got := strings.Count(message, "too complex"); got != 5 {
		t.Fatalf("expected 5 listed findings, got %d:\n%s", got, message)
	}
	if !strings.Contains(message, "and 2 more blocking findings") {
		t.Fatalf("feedback must say what it left out:\n%s", message)
	}
	if !strings.Contains(message, "`C:/tools/lumioguard-cc.exe check --base HEAD --format json`") {
		t.Fatalf("feedback must name the command as the hook was started:\n%s", message)
	}
}

func TestInvokedAs(t *testing.T) {
	for arg0, want := range map[string]string{
		"lumioguard-cc":                "lumioguard-cc",
		"lumioguard-cc.exe":            "lumioguard-cc",
		"/usr/local/bin/lumioguard-cc": "/usr/local/bin/lumioguard-cc",
		"/home/a b/bin/lumioguard-cc":  `"/home/a b/bin/lumioguard-cc"`,
		"./bin/lumioguard-cc":          "./bin/lumioguard-cc",
	} {
		if got := invokedAs(arg0); got != want {
			t.Errorf("invokedAs(%q) = %q, want %q", arg0, got, want)
		}
	}
}
