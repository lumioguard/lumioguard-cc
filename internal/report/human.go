package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/product"
)

// findingLimit caps how many findings and diagnostics a terminal summary lists.
const findingLimit = 10

// HumanRenderer writes a short prioritised summary for a terminal.
type HumanRenderer struct{}

// Render implements Renderer.
func (r HumanRenderer) Render(w io.Writer, report *domain.Report) error {
	_, err := io.WriteString(w, r.Text(report)+"\n")
	return err
}

// Text builds the summary without writing it.
func (r HumanRenderer) Text(report *domain.Report) string {
	lines := []string{
		fmt.Sprintf("%s check: %s", product.DisplayName, strings.ToUpper(string(report.Policy.Status))),
		fmt.Sprintf("%d files analyzed in %d ms", report.Scope.FilesAnalyzed, report.Run.DurationMs),
		fmt.Sprintf("%d blocking, %d advisory findings", report.Policy.BlockingFindings, report.Policy.WarningFindings),
	}
	if line, ok := tokenLine(report); ok {
		lines = append(lines, line)
	}
	active := report.ActiveFindings()
	for _, finding := range truncate(active, findingLimit) {
		lines = append(lines, fmt.Sprintf("- [%s] %s: %s (%s)", finding.Severity, finding.Scope.Location(), finding.Message, finding.Classification))
	}
	if resolved := report.ResolvedCount(); resolved > 0 {
		lines = append(lines, fmt.Sprintf("Resolved since %s: %d", comparisonNoun(report.Comparison), resolved))
	}
	var notable []domain.Diagnostic
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Severity != domain.SeverityInfo {
			notable = append(notable, diagnostic)
		}
	}
	for _, diagnostic := range truncate(notable, findingLimit) {
		lines = append(lines, fmt.Sprintf("- [%s] %s: %s", diagnostic.Severity, diagnostic.Code, diagnostic.Message))
	}
	if len(active) == 0 && report.Policy.Status != domain.PolicyIncomplete {
		lines = append(lines, "No active findings exceeded the configured policy.")
	}
	return strings.Join(lines, "\n")
}

// tokenLine reports how much code there is to read, and how much the change
// added, because that number is the cost every later task pays. An incomplete
// run has not read everything, so its total is not shown.
func tokenLine(report *domain.Report) (string, bool) {
	if report.Policy.Status == domain.PolicyIncomplete {
		return "", false
	}
	for _, measurement := range report.Measurements {
		if measurement.MetricID != domain.MetricTotalTokens || !measurement.IsMeasured() {
			continue
		}
		line := "Source tokens: " + domain.FormatNumber(*measurement.Value)
		switch {
		case measurement.Delta == nil:
		case *measurement.Delta == 0:
			line += fmt.Sprintf(" (no change since %s)", comparisonNoun(report.Comparison))
		default:
			delta := domain.FormatNumber(*measurement.Delta)
			if *measurement.Delta > 0 {
				delta = "+" + delta
			}
			line += fmt.Sprintf(" (%s since %s)", delta, comparisonNoun(report.Comparison))
		}
		return line, true
	}
	return "", false
}

// comparisonNoun names what the report was compared with. A Git reference is a
// full commit hash, too long for a summary line, so the mode is named instead.
func comparisonNoun(comparison domain.Comparison) string {
	if comparison.Mode == domain.ComparisonGit {
		return "the base commit"
	}
	return "baseline"
}

func truncate[T any](items []T, limit int) []T {
	if len(items) > limit {
		return items[:limit]
	}
	return items
}
