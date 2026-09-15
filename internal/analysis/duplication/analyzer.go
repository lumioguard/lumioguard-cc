// Package duplication measures exact token-clone density: the share of
// nonblank source lines that participate in a repeated window of tokens.
package duplication

import (
	"context"
	"fmt"

	"github.com/lumiostack/lumioguard-cc/internal/analysis"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/fingerprint"
)

// Name identifies the analyzer in error messages.
const Name = "token-duplication"

// Analyzer implements analysis.RepositoryAnalyzer for exact token clones.
type Analyzer struct {
	recheck string
}

// New creates the analyzer; recheck is the command shown in findings.
func New(recheck string) *Analyzer {
	return &Analyzer{recheck: recheck}
}

// Name implements analysis.RepositoryAnalyzer.
func (a *Analyzer) Name() string {
	return Name
}

// Analyze implements analysis.RepositoryAnalyzer.
func (a *Analyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	policy := in.Config.Metrics.DuplicationPercent
	detector := newDetector(policy.MinTokens, policy.MinLines)
	groups, duplicatedLines := detector.detect(in.Files)

	totalNonblank := 0
	for _, file := range in.Files {
		totalNonblank += file.NonBlankLineCount()
	}
	duplicateLineCount := 0
	for _, lines := range duplicatedLines {
		duplicateLineCount += len(lines)
	}

	measurement := domain.Measurement{
		MetricID: domain.MetricTokenCloneDensity,
		Variant:  fmt.Sprintf("exact-token-window-v1:%d-tokens:%d-lines", policy.MinTokens, policy.MinLines),
		Category: domain.CategoryDuplication,
		Scope:    domain.RepositoryScope(),
		Unit:     domain.UnitPercent,
		Status:   domain.StatusMeasured,
		Evidence: domain.Evidence{
			"duplicateLineCount": duplicateLineCount,
			"totalNonblankLines": totalNonblank,
			"cloneGroups":        len(groups),
		},
	}
	if totalNonblank == 0 {
		measurement.Status = domain.StatusNotApplicable
		measurement.Reason = "No eligible source lines"
	} else {
		measurement.Value = domain.Float(domain.Round(float64(duplicateLineCount)/float64(totalNonblank)*100, 2))
	}

	result := analysis.Result{Measurements: []domain.Measurement{measurement}}
	if measurement.Value != nil && policy.Exceeds(*measurement.Value) {
		result.Findings = a.cloneFindings(groups, policy)
	}
	return result, nil
}

func (a *Analyzer) cloneFindings(groups []cloneGroup, policy domain.DuplicationPolicy) []domain.Finding {
	findings := make([]domain.Finding, 0, len(groups))
	for _, group := range groups {
		findings = append(findings, domain.Finding{
			ID:             fingerprint.ShortID(string(domain.RuleTokenClone), group.Fingerprint),
			RuleID:         domain.RuleTokenClone,
			Title:          "Duplicated token block",
			Severity:       policy.Severity,
			Blocking:       policy.Block,
			Scope:          domain.Scope{Kind: domain.ScopeRepository, Key: "clone:" + group.Fingerprint},
			Message:        fmt.Sprintf("The same %d-token block appears in %d locations", policy.MinTokens, len(group.Occurrences)),
			Classification: domain.ClassificationNew,
			Current:        domain.Float(float64(len(group.Occurrences))),
			Unit:           domain.UnitCount,
			Evidence:       domain.Evidence{"fingerprint": group.Fingerprint, "occurrences": group.Occurrences},
			Recheck:        a.recheck,
		})
	}
	return findings
}
