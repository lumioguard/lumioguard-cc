// Package comparison classifies a report against a baseline. It compares only
// when the schema, adapter identities and configuration hash all match.
package comparison

import (
	"fmt"
	"maps"
	"slices"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

// Comparer implements the baseline comparison rules.
type Comparer struct{}

// New creates a Comparer.
func New() *Comparer {
	return &Comparer{}
}

// Compare returns a copy of report with baseline values, deltas and classifications.
// An incompatible baseline adds a required diagnostic and classifies nothing.
func (c *Comparer) Compare(report *domain.Report, baseline *domain.Baseline, mode domain.ComparisonMode, reference string) *domain.Report {
	out := *report
	out.Diagnostics = slices.Clone(report.Diagnostics)
	out.Comparison = domain.Comparison{Mode: mode, Reference: &reference, Comparable: false}

	if code, message, incompatible := c.incompatibility(report, baseline); incompatible {
		out.Diagnostics = append(out.Diagnostics, domain.RequiredError(code, "", message))
		return &out
	}

	renames := renameMap(report.Snapshot.FileHashes, baseline.Snapshot.FileHashes)
	out.Measurements = compareMeasurements(report.Measurements, baseline.Measurements, renames)
	out.Findings = compareFindings(report.Findings, baseline.Findings, out.Measurements, renames)
	out.Comparison.Comparable = true
	return &out
}

func (c *Comparer) incompatibility(report *domain.Report, baseline *domain.Baseline) (code, message string, incompatible bool) {
	switch {
	case baseline.SchemaVersion != domain.BaselineSchemaVersion:
		return "baseline.schema_incompatible", fmt.Sprintf("Unsupported baseline schema %s", baseline.SchemaVersion), true
	case !sameAdapterIdentities(baseline.Adapters, report.Adapters):
		return "baseline.analyzers_incompatible",
			"Language adapter or analyzer versions differ from the baseline; create a reviewed replacement baseline", true
	case baseline.Snapshot.ConfigHash != report.Snapshot.ConfigHash:
		return "baseline.config_incompatible",
			"The configuration differs from the baseline; create a reviewed replacement baseline before comparing", true
	default:
		return "", "", false
	}
}

// sameAdapterIdentities ignores per-run counts such as filesAnalyzed, which
// change between runs and would otherwise make every baseline incompatible.
func sameAdapterIdentities(a, b []domain.AdapterSummary) bool {
	return slices.EqualFunc(a, b, func(left, right domain.AdapterSummary) bool {
		return left.ID == right.ID &&
			left.Version == right.Version &&
			slices.Equal(left.Languages, right.Languages) &&
			maps.Equal(left.Analyzers, right.Analyzers)
	})
}

func measurementKey(m domain.Measurement, scope domain.Scope) string {
	return string(m.MetricID) + "\x00" + m.Variant + "\x00" + scope.Key
}

func findingKey(f domain.Finding, scope domain.Scope) string {
	return string(f.RuleID) + "\x00" + scope.Key
}

func compareMeasurements(current, previous []domain.Measurement, renames map[string]string) []domain.Measurement {
	index := make(map[string]domain.Measurement, len(previous))
	for _, measurement := range previous {
		index[measurementKey(measurement, remap(measurement.Scope, renames))] = measurement
	}
	result := make([]domain.Measurement, len(current))
	for i, measurement := range current {
		result[i] = measurement
		before, ok := index[measurementKey(measurement, measurement.Scope)]
		if !ok || !before.IsMeasured() || !measurement.IsMeasured() {
			continue
		}
		delta := *measurement.Value - *before.Value
		result[i].Baseline = domain.Float(*before.Value)
		result[i].Delta = domain.Float(delta)
		result[i].Classification = classifyDelta(delta)
	}
	return result
}

func classifyDelta(delta float64) domain.Classification {
	switch {
	case delta > 0:
		return domain.ClassificationWorsened
	case delta < 0:
		return domain.ClassificationImproved
	default:
		return domain.ClassificationExisting
	}
}

func compareFindings(current, previous []domain.Finding, measurements []domain.Measurement, renames map[string]string) []domain.Finding {
	measurementIndex := make(map[string]domain.Measurement, len(measurements))
	for _, measurement := range measurements {
		measurementIndex[measurementKey(measurement, measurement.Scope)] = measurement
	}
	previousIndex := make(map[string]domain.Finding, len(previous))
	previousOrder := make([]string, 0, len(previous))
	for _, finding := range previous {
		scope := remap(finding.Scope, renames)
		finding.Scope = scope
		key := findingKey(finding, scope)
		previousIndex[key] = finding
		previousOrder = append(previousOrder, key)
	}

	seen := make(map[string]struct{}, len(current))
	result := make([]domain.Finding, 0, len(current)+len(previous))
	for _, finding := range current {
		key := findingKey(finding, finding.Scope)
		seen[key] = struct{}{}
		before, existed := previousIndex[key]
		variant, _ := finding.Evidence["metricVariant"].(string)
		measurement, measured := measurementIndex[string(finding.RuleID)+"\x00"+variant+"\x00"+finding.Scope.Key]

		var baselineValue *float64
		switch {
		case measured && measurement.Baseline != nil:
			baselineValue = domain.Float(*measurement.Baseline)
		case existed && before.Current != nil:
			baselineValue = domain.Float(*before.Current)
		}
		var delta *float64
		if finding.Current != nil && baselineValue != nil {
			delta = domain.Float(*finding.Current - *baselineValue)
		}
		finding.Baseline = baselineValue
		finding.Delta = delta
		switch {
		case !existed:
			finding.Classification = domain.ClassificationNew
		case delta != nil && *delta > 0:
			finding.Classification = domain.ClassificationWorsened
		default:
			finding.Classification = domain.ClassificationExisting
		}
		result = append(result, finding)
	}
	for _, key := range previousOrder {
		if _, still := seen[key]; still {
			continue
		}
		before := previousIndex[key]
		resolved := before
		resolved.Classification = domain.ClassificationResolved
		resolved.Baseline = before.Current
		resolved.Current = nil
		resolved.Delta = nil
		if before.Current != nil {
			resolved.Delta = domain.Float(-*before.Current)
		}
		resolved.Blocking = false
		resolved.Message = "Resolved: " + before.Message
		result = append(result, resolved)
	}
	return result
}

func remap(scope domain.Scope, renames map[string]string) domain.Scope {
	if scope.File == "" {
		return scope
	}
	if moved, ok := renames[scope.File]; ok {
		return scope.WithFile(moved)
	}
	return scope
}
