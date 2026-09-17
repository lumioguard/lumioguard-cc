package worklist

import (
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

var (
	cognitive  = domain.RuleID(domain.MetricCognitive)
	cyclomatic = domain.RuleID(domain.MetricCyclomatic)
	lines      = domain.RuleID(domain.MetricFunctionLines)
)

// limits are the thresholds the test findings exceed.
var limits = map[domain.RuleID]float64{cognitive: 15, cyclomatic: 12, lines: 80}

func at(file, symbol string, line int) domain.Scope {
	return domain.FunctionScope(file, symbol, line, line+10)
}

func breach(scope domain.Scope, rule domain.RuleID, current float64) domain.Finding {
	return domain.Finding{
		RuleID:         rule,
		Blocking:       true,
		Scope:          scope,
		Classification: domain.ClassificationExisting,
		Current:        domain.Float(current),
		Threshold:      domain.Float(limits[rule]),
		Unit:           domain.UnitCount,
	}
}

func report(findings ...domain.Finding) *domain.Report {
	return &domain.Report{Findings: findings, Comparison: domain.Comparison{Mode: domain.ComparisonNone}}
}

func TestHotspotsBreakingMoreRulesComeFirstAndNestedFunctionsFold(t *testing.T) {
	// Findings arrive sorted by rule, so a nested function can precede its outer one.
	list := Build(report(
		breach(at("a.ts", "outer.inner.deep", 4), cognitive, 16),
		breach(at("a.ts", "outer.second", 8), cognitive, 17),
		breach(at("b.ts", "small", 1), cognitive, 40),
		breach(at("a.ts", "outer", 1), cognitive, 20),
		breach(at("c.ts", "lonely.child", 2), cognitive, 16),
		breach(at("a.ts", "outer.inner", 3), cyclomatic, 13),
		breach(at("a.ts", "outer", 1), lines, 90),
	), Options{})
	if len(list.Hotspots) != 3 || list.Findings != 7 {
		t.Fatalf("expected outer, small and lonely.child from seven findings, got %+v", list)
	}
	outer := list.Hotspots[0]
	if outer.Scope.Symbol != "outer" || outer.Rules != 3 || len(outer.Nested) != 2 {
		t.Fatalf("outer must come first with three rules and two nested functions: %+v", outer)
	}
	if outer.Nested[0].Scope.Symbol != "outer.inner" || outer.Nested[1].Scope.Symbol != "outer.second" {
		t.Fatalf("nested functions are in line order: %+v", outer.Nested)
	}
	if len(outer.Nested[0].Nested) != 1 || outer.Nested[0].Nested[0].Scope.Symbol != "outer.inner.deep" {
		t.Fatalf("deep must fold into inner: %+v", outer.Nested[0])
	}
	if list.Hotspots[1].Scope.Symbol != "small" || list.Hotspots[2].Scope.Symbol != "lonely.child" {
		t.Fatalf("one-rule entries follow, the one furthest over its limit first: %+v", list.Hotspots[1:])
	}
}

func mixedFindings() []domain.Finding {
	clone := domain.Finding{
		RuleID: domain.RuleTokenClone, Scope: domain.Scope{Kind: domain.ScopeRepository, Key: "clone:a.ts|z.ts"},
		Current: domain.Float(40), Unit: domain.UnitLines,
		Evidence: domain.Evidence{"occurrences": []map[string]any{{"file": "a.ts", "startLine": 1, "endLine": 20}, {"file": "z.ts", "startLine": 5, "endLine": 24}}},
	}
	cycle := domain.Finding{RuleID: domain.RuleDependencyCycle, Scope: domain.CycleScope("a.ts|z.ts"), Evidence: domain.Evidence{"modules": []string{"a.ts", "z.ts"}}}
	density := domain.Finding{RuleID: domain.RuleID(domain.MetricTokenCloneDensity), Scope: domain.RepositoryScope(), Current: domain.Float(12), Threshold: domain.Float(5), Unit: domain.UnitPercent}
	resolved := breach(at("a.ts", "gone", 1), cognitive, 20)
	resolved.Classification = domain.ClassificationResolved
	return []domain.Finding{
		clone, cycle, density, resolved,
		breach(at("a.ts", "one", 1), cognitive, 20),
		breach(at("b.ts", "two", 1), cognitive, 30),
		breach(at("b.ts", "three", 1), cyclomatic, 13),
	}
}

func TestRuleFilterKeepsOneRuleAndNeverAResolvedFinding(t *testing.T) {
	list := Build(report(mixedFindings()...), Options{Rules: []domain.RuleID{cognitive}})
	if list.Findings != 2 || len(list.Hotspots) != 2 || len(list.Clones) != 0 || len(list.Structure) != 0 || list.Density != nil {
		t.Fatalf("--rule keeps only that rule: %+v", list)
	}
}

func TestPathFilterMatchesAnyFileACloneOrCycleNames(t *testing.T) {
	list := Build(report(mixedFindings()...), Options{Paths: []string{"z.ts"}})
	if list.Findings != 2 || len(list.Clones) != 1 || len(list.Structure) != 1 || len(list.Hotspots) != 0 {
		t.Fatalf("--path matches any file a clone or cycle names: %+v", list)
	}
	if list.Clones[0].Copies[1].File != "z.ts" {
		t.Fatalf("clone entries list their copies: %+v", list.Clones[0])
	}
}

func TestTopCapsEachSectionAndDensityHeadsTheClones(t *testing.T) {
	list := Build(report(mixedFindings()...), Options{Top: 1})
	if len(list.Hotspots) != 1 || list.Hotspots[0].Scope.Symbol != "two" || list.Omitted != 2 {
		t.Fatalf("--top keeps the first entries of each section and counts the rest: %+v", list)
	}
	if list.Density == nil || *list.Density.Current != 12 || list.Findings != 6 {
		t.Fatalf("the density finding heads the duplication section instead of being a hotspot: %+v", list)
	}
}
