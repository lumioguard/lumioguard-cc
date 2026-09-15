// Package policy turns measurements into threshold findings and computes the gate.
// Only new or worsened blocking findings fail it, and incomplete wins over failed.
package policy

import (
	"fmt"

	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/fingerprint"
)

// Evaluator applies configured metric policies.
type Evaluator struct {
	recheck string
}

// New creates an Evaluator; recheck is the command shown in findings.
func New(recheck string) *Evaluator {
	return &Evaluator{recheck: recheck}
}

// ThresholdFindings creates one finding per measured value above its policy threshold.
func (e *Evaluator) ThresholdFindings(measurements []domain.Measurement, cfg domain.Config) []domain.Finding {
	var findings []domain.Finding
	for _, measurement := range measurements {
		policy, governed := cfg.PolicyFor(measurement.MetricID)
		if !governed || !measurement.IsMeasured() || !policy.Exceeds(*measurement.Value) {
			continue
		}
		evidence := domain.Evidence{"metricVariant": measurement.Variant}
		for key, value := range measurement.Evidence {
			evidence[key] = value
		}
		findings = append(findings, domain.Finding{
			ID:       fingerprint.ShortID(string(measurement.MetricID), measurement.Scope.Key),
			RuleID:   domain.RuleID(measurement.MetricID),
			Title:    fmt.Sprintf("%s exceeds project threshold", measurement.MetricID),
			Severity: policy.Severity,
			Blocking: policy.Block,
			Scope:    measurement.Scope,
			Message: fmt.Sprintf("%s is %s %s; the configured threshold is %s",
				measurement.MetricID, domain.FormatNumber(*measurement.Value), measurement.Unit, domain.FormatNumber(policy.Threshold)),
			Classification: domain.ClassificationNew,
			Current:        domain.Float(*measurement.Value),
			Threshold:      domain.Float(policy.Threshold),
			Unit:           measurement.Unit,
			Evidence:       evidence,
			Recheck:        e.recheck,
		})
	}
	return findings
}

// Result computes the gate outcome from classified findings.
func (e *Evaluator) Result(findings []domain.Finding, incomplete bool) domain.PolicyResult {
	result := domain.PolicyResult{Status: domain.PolicyPassed}
	for _, finding := range findings {
		switch {
		case finding.BlocksGate():
			result.BlockingFindings++
		case !finding.Blocking && finding.IsActive():
			result.WarningFindings++
		}
	}
	switch {
	case incomplete:
		result.Status = domain.PolicyIncomplete
	case result.BlockingFindings > 0:
		result.Status = domain.PolicyFailed
	}
	return result
}
