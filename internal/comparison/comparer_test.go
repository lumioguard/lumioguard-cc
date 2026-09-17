package comparison

import (
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

func measurement(file, symbol string, value float64) domain.Measurement {
	return domain.Measurement{
		MetricID: domain.MetricCyclomatic,
		Variant:  "v1",
		Category: domain.CategoryComplexity,
		Scope:    domain.FunctionScope(file, symbol, 1, 3),
		Value:    domain.Float(value),
		Unit:     domain.UnitCount,
		Status:   domain.StatusMeasured,
	}
}

func finding(file, symbol string, value float64) domain.Finding {
	return domain.Finding{
		ID:             "id-" + symbol,
		RuleID:         domain.RuleID(domain.MetricCyclomatic),
		Title:          "t",
		Severity:       domain.SeverityWarning,
		Blocking:       true,
		Scope:          domain.FunctionScope(file, symbol, 1, 3),
		Message:        "m",
		Classification: domain.ClassificationNew,
		Current:        domain.Float(value),
		Threshold:      domain.Float(1),
		Unit:           domain.UnitCount,
		Evidence:       domain.Evidence{"metricVariant": "v1"},
	}
}

func report(hashes map[string]string, measurements []domain.Measurement, findings []domain.Finding) *domain.Report {
	return &domain.Report{
		Snapshot:     domain.Snapshot{FileHashes: hashes, ConfigHash: "cfg"},
		Adapters:     []domain.AdapterSummary{{ID: "js", Version: "1", Languages: []string{"js"}, Analyzers: map[string]string{"p": "1"}, FilesAnalyzed: len(hashes)}},
		Measurements: measurements,
		Findings:     findings,
		Diagnostics:  []domain.Diagnostic{},
	}
}

func baselineOf(r *domain.Report) *domain.Baseline {
	return &domain.Baseline{
		SchemaVersion: domain.BaselineSchemaVersion,
		Snapshot:      r.Snapshot,
		Adapters:      r.Adapters,
		Measurements:  r.Measurements,
		Findings:      r.Findings,
	}
}

func TestCompareClassifiesWorsenedMeasurementsAndFindings(t *testing.T) {
	before := report(map[string]string{"a.ts": "h1"}, []domain.Measurement{measurement("a.ts", "check", 2)}, []domain.Finding{finding("a.ts", "check", 2)})
	current := report(map[string]string{"a.ts": "h2"}, []domain.Measurement{measurement("a.ts", "check", 3)}, []domain.Finding{finding("a.ts", "check", 3)})

	compared := New().Compare(current, baselineOf(before), domain.ComparisonBaseline, "initial")
	if !compared.Comparison.Comparable || compared.Comparison.Mode != domain.ComparisonBaseline {
		t.Fatalf("expected comparable baseline comparison, got %+v", compared.Comparison)
	}
	m := compared.Measurements[0]
	if m.Baseline == nil || *m.Baseline != 2 || m.Delta == nil || *m.Delta != 1 || m.Classification != domain.ClassificationWorsened {
		t.Fatalf("measurement not classified as worsened: %+v", m)
	}
	f := compared.Findings[0]
	if f.Baseline == nil || *f.Baseline != 2 || f.Delta == nil || *f.Delta != 1 || f.Classification != domain.ClassificationWorsened {
		t.Fatalf("finding not classified as worsened: %+v", f)
	}
}

func TestCompareKeepsUnchangedFindingsExistingAndResolvesMissingOnes(t *testing.T) {
	before := report(map[string]string{"a.ts": "h1"}, []domain.Measurement{measurement("a.ts", "check", 2)}, []domain.Finding{finding("a.ts", "check", 2), finding("a.ts", "gone", 5)})
	current := report(map[string]string{"a.ts": "h1"}, []domain.Measurement{measurement("a.ts", "check", 2)}, []domain.Finding{finding("a.ts", "check", 2)})

	compared := New().Compare(current, baselineOf(before), domain.ComparisonBaseline, "initial")
	if compared.Findings[0].Classification != domain.ClassificationExisting {
		t.Fatalf("expected existing, got %s", compared.Findings[0].Classification)
	}
	resolved := compared.Findings[1]
	if resolved.Classification != domain.ClassificationResolved || resolved.Blocking || resolved.Current != nil || resolved.Baseline == nil || *resolved.Baseline != 5 {
		t.Fatalf("expected resolved finding, got %+v", resolved)
	}
	if compared.Measurements[0].Classification != domain.ClassificationExisting {
		t.Fatalf("expected existing measurement, got %s", compared.Measurements[0].Classification)
	}
}

func TestCompareRefusesIncompatibleBaselines(t *testing.T) {
	before := report(map[string]string{"a.ts": "h1"}, nil, nil)
	current := report(map[string]string{"a.ts": "h1"}, nil, nil)
	current.Snapshot.ConfigHash = "other"
	compared := New().Compare(current, baselineOf(before), domain.ComparisonBaseline, "initial")
	if compared.Comparison.Comparable {
		t.Fatal("expected incomparable comparison")
	}
	if !domain.HasRequiredError(compared.Diagnostics) || compared.Diagnostics[0].Code != "baseline.config_incompatible" {
		t.Fatalf("expected config incompatibility diagnostic, got %+v", compared.Diagnostics)
	}

	current.Snapshot.ConfigHash = "cfg"
	stale := baselineOf(before)
	stale.SchemaVersion = "0.9"
	if compared := New().Compare(current, stale, domain.ComparisonBaseline, "x"); compared.Diagnostics[0].Code != "baseline.schema_incompatible" {
		t.Fatalf("expected schema diagnostic, got %+v", compared.Diagnostics)
	}
	other := baselineOf(before)
	other.Adapters = []domain.AdapterSummary{{ID: "js", Version: "2"}}
	if compared := New().Compare(current, other, domain.ComparisonBaseline, "x"); compared.Diagnostics[0].Code != "baseline.analyzers_incompatible" {
		t.Fatalf("expected analyzer diagnostic, got %+v", compared.Diagnostics)
	}
}

func TestCompareIgnoresPerRunAdapterCounts(t *testing.T) {
	before := report(map[string]string{"a.ts": "h1"}, nil, nil)
	current := report(map[string]string{"a.ts": "h1", "b.ts": "h2"}, nil, nil)
	compared := New().Compare(current, baselineOf(before), domain.ComparisonBaseline, "initial")
	if !compared.Comparison.Comparable {
		t.Fatalf("adding a file must not make the baseline incompatible: %+v", compared.Diagnostics)
	}
}

// The report schema requires arrays, so an empty list must not become JSON null.
func TestCompareKeepsEmptyListsNonNil(t *testing.T) {
	before := report(map[string]string{"a.ts": "h1"}, nil, nil)
	compared := New().Compare(report(map[string]string{"a.ts": "h1"}, nil, nil), baselineOf(before), domain.ComparisonGit, "abc")
	if compared.Diagnostics == nil || compared.Findings == nil || compared.Measurements == nil {
		t.Fatalf("empty lists became nil: diagnostics=%v findings=%v measurements=%v", compared.Diagnostics == nil, compared.Findings == nil, compared.Measurements == nil)
	}
}

func TestCompareFollowsUnchangedContentRenames(t *testing.T) {
	before := report(map[string]string{"old.ts": "same"}, []domain.Measurement{measurement("old.ts", "check", 4)}, []domain.Finding{finding("old.ts", "check", 4)})
	current := report(map[string]string{"new.ts": "same"}, []domain.Measurement{measurement("new.ts", "check", 4)}, []domain.Finding{finding("new.ts", "check", 4)})
	compared := New().Compare(current, baselineOf(before), domain.ComparisonBaseline, "initial")
	if compared.Findings[0].Classification != domain.ClassificationExisting {
		t.Fatalf("renamed file finding should be existing, got %+v", compared.Findings[0])
	}
	if len(compared.Findings) != 1 {
		t.Fatalf("renamed file must not produce a resolved finding: %+v", compared.Findings)
	}
}

func TestRenameMapIgnoresAmbiguousMatches(t *testing.T) {
	renames := renameMap(map[string]string{"x.ts": "h", "y.ts": "h"}, map[string]string{"old.ts": "h"})
	if len(renames) != 0 {
		t.Fatalf("ambiguous rename must not be guessed: %v", renames)
	}
}
