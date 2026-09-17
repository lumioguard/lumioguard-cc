// Package duplication measures exact token-clone density: the share of
// nonblank source lines that participate in a repeated window of tokens.
package duplication

import (
	"context"
	"fmt"
	"slices"
	"strings"

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
		Variant:  fmt.Sprintf("exact-token-region-v2:%d-tokens:%d-lines", policy.MinTokens, policy.MinLines),
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

// cloneFindings reports one finding per region. Its identity is the set of
// files and functions holding the copies, so it survives edits to the copied
// text, and its value is the number of lines the copies occupy in all.
func (a *Analyzer) cloneFindings(groups []cloneGroup, policy domain.DuplicationPolicy) []domain.Finding {
	findings := make([]domain.Finding, 0, len(groups))
	ordinals := make(map[string]int, len(groups))
	for _, group := range groups {
		key := memberKey(group)
		ordinals[key]++
		if ordinal := ordinals[key]; ordinal > 1 {
			key = fmt.Sprintf("%s#%d", key, ordinal)
		}
		lines := 0
		for _, item := range group.Occurrences {
			lines += item.EndLine - item.StartLine + 1
		}
		findings = append(findings, domain.Finding{
			ID:       fingerprint.ShortID(string(domain.RuleTokenClone), key),
			RuleID:   domain.RuleTokenClone,
			Title:    "Duplicated block",
			Severity: policy.Severity,
			Blocking: policy.Block,
			Scope:    domain.Scope{Kind: domain.ScopeRepository, Key: key},
			Message: fmt.Sprintf("The same %d-token block appears in %d locations, %d lines in all",
				group.Tokens, len(group.Occurrences), lines),
			Classification: domain.ClassificationNew,
			Current:        domain.Float(float64(lines)),
			Unit:           domain.UnitLines,
			Evidence: domain.Evidence{
				"fingerprint": group.Fingerprint,
				"tokens":      group.Tokens,
				"occurrences": group.Occurrences,
			},
			Recheck: a.recheck,
		})
	}
	return findings
}

// memberKey names a group by where its copies live: each file, with the
// function when the copy is inside one, sorted and joined.
func memberKey(group cloneGroup) string {
	members := make([]string, 0, len(group.Occurrences))
	for _, item := range group.Occurrences {
		member := item.File
		if item.Symbol != "" {
			member += "::" + item.Symbol
		}
		members = append(members, member)
	}
	slices.Sort(members)
	return "clone:" + strings.Join(members, "|")
}
