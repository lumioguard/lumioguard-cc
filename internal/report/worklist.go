package report

import (
	"fmt"
	"strings"

	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/product"
	"github.com/lumiostack/lumioguard-cc/internal/worklist"
)

// WorklistText renders a worklist for a terminal: one line per place, with
// the rules it breaks, nested functions indented under their function.
func WorklistText(list *worklist.Worklist) string {
	places := len(list.Structure) + len(list.Hotspots) + len(list.Clones) + list.Omitted
	lines := []string{fmt.Sprintf("%s worklist: %d findings in %d places%s", product.DisplayName, list.Findings, places, comparedWith(list.Comparison))}
	if list.Policy.Status == domain.PolicyIncomplete {
		lines = append(lines, "Analysis is incomplete: some code was not analyzed, so this list is missing findings.")
	}
	compared := list.Comparison.Mode != domain.ComparisonNone
	if len(list.Structure) > 0 {
		lines = append(lines, "Structure, fix first:")
		for _, entry := range list.Structure {
			lines = append(lines, fmt.Sprintf("- %s: %s%s", entry.Scope.Location(), entry.Findings[0].Message, classified(entry.Findings[0], compared)))
		}
	}
	if len(list.Hotspots) > 0 {
		lines = append(lines, "Hotspots, most rules broken first:")
		for index, entry := range list.Hotspots {
			lines = append(lines, fmt.Sprintf("%d. %s%s", index+1, describe(entry), classifiedAll(entry, compared)))
			lines = append(lines, nestedLines(entry.Nested, "   ")...)
		}
	}
	if len(list.Clones) > 0 || list.Density != nil {
		header := "Duplicated blocks, largest first"
		if list.Density != nil {
			header += fmt.Sprintf(" (%s%s)", ruleValue(*list.Density), classified(*list.Density, compared))
		}
		lines = append(lines, header+":")
		for _, entry := range list.Clones {
			lines = append(lines, fmt.Sprintf("- %s%s", cloneLine(entry), classified(entry.Findings[0], compared)))
		}
	}
	if list.Omitted > 0 {
		lines = append(lines, fmt.Sprintf("%d more places not shown; use --top 0 to see all", list.Omitted))
	}
	if places == 0 {
		lines = append(lines, "Nothing to fix.")
	}
	return strings.Join(lines, "\n")
}

func comparedWith(comparison domain.Comparison) string {
	if comparison.Mode == domain.ComparisonNone {
		return ""
	}
	return ", compared with " + comparisonNoun(comparison)
}

// describe renders "file:line symbol  rule value (limit n), ..." for a place.
func describe(entry worklist.Entry) string {
	where := entry.Scope.Location()
	if entry.Scope.Symbol != "" {
		where += " " + entry.Scope.Symbol
	}
	parts := make([]string, 0, len(entry.Findings))
	for _, finding := range entry.Findings {
		parts = append(parts, ruleValue(finding))
	}
	return where + "  " + strings.Join(parts, ", ")
}

func ruleValue(finding worklist.Item) string {
	text := string(finding.RuleID)
	if finding.Current != nil {
		text += " " + domain.FormatNumber(*finding.Current)
	}
	if finding.Threshold != nil {
		text += fmt.Sprintf(" (limit %s)", domain.FormatNumber(*finding.Threshold))
	}
	return text
}

func nestedLines(nested []worklist.Entry, indent string) []string {
	var lines []string
	for _, entry := range nested {
		lines = append(lines, indent+"+ "+describe(entry))
		lines = append(lines, nestedLines(entry.Nested, indent+"  ")...)
	}
	return lines
}

func cloneLine(entry worklist.Entry) string {
	places := make([]string, 0, len(entry.Copies))
	for _, copy := range entry.Copies {
		place := fmt.Sprintf("%s:%d-%d", copy.File, copy.StartLine, copy.EndLine)
		if copy.Symbol != "" {
			place += " " + copy.Symbol
		}
		places = append(places, place)
	}
	lines := "?"
	if entry.Findings[0].Current != nil {
		lines = domain.FormatNumber(*entry.Findings[0].Current)
	}
	return fmt.Sprintf("%s lines in %d places: %s", lines, len(entry.Copies), strings.Join(places, ", "))
}

// classified appends the classification when the list was compared with something.
func classified(finding worklist.Item, compared bool) string {
	if !compared {
		return ""
	}
	return fmt.Sprintf(" (%s)", finding.Classification)
}

// classifiedAll appends the strongest classification among a place's findings.
func classifiedAll(entry worklist.Entry, compared bool) string {
	if !compared {
		return ""
	}
	strongest := domain.ClassificationExisting
	for _, finding := range entry.Findings {
		if finding.Classification == domain.ClassificationNew || (finding.Classification == domain.ClassificationWorsened && strongest != domain.ClassificationNew) {
			strongest = finding.Classification
		}
	}
	return fmt.Sprintf(" (%s)", strongest)
}
