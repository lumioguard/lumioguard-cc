package cli_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

const nestedSample = `export function outer(a: boolean, b: boolean) {
  const inner = (x: boolean) => (x ? 1 : a ? 2 : 3);
  if (a && b) { return inner(a); }
  return inner(b);
}
export function simple(flag: boolean) { return flag ? 1 : 0; }
`

func worklistProject(t *testing.T) string {
	t.Helper()
	cfg := blockingCyclomatic()
	cfg.Metrics.Cognitive.Threshold = 1
	return project(t, nestedSample, cfg)
}

type worklistJSON struct {
	Findings int `json:"findings"`
	Hotspots []struct {
		Scope    domain.Scope `json:"scope"`
		Rules    int          `json:"rules"`
		Findings []struct {
			RuleID domain.RuleID `json:"ruleId"`
		} `json:"findings"`
		Nested []struct {
			Scope domain.Scope `json:"scope"`
		} `json:"nested"`
	} `json:"hotspots"`
	Omitted int `json:"omitted"`
}

func TestWorklistListsTheOuterFunctionFirstWithNestedFunctionsUnderIt(t *testing.T) {
	human := execute(t, "", "worklist", "--root", worklistProject(t))
	if human.code != 0 || !strings.Contains(human.stdout, "Hotspots") {
		t.Fatalf("worklist must succeed and list hotspots: %+v", human)
	}
	if !strings.Contains(human.stdout, "1. sample.ts:1 outer") || !strings.Contains(human.stdout, "+ sample.ts:2 outer.inner") {
		t.Fatalf("outer must come first with inner folded under it:\n%s", human.stdout)
	}
}

func TestWorklistFiltersByRuleAndCapsEachSection(t *testing.T) {
	structured := execute(t, "", "worklist", "--root", worklistProject(t), "--format", "json", "--rule", "complexity.cyclomatic", "--top", "1")
	var list worklistJSON
	if err := json.Unmarshal([]byte(structured.stdout), &list); err != nil {
		t.Fatalf("worklist JSON: %v\n%s", err, structured.stdout)
	}
	if structured.code != 0 || list.Findings != 3 || len(list.Hotspots) != 1 || list.Omitted != 1 {
		t.Fatalf("--rule and --top must filter and cap: %+v", list)
	}
	outer := list.Hotspots[0]
	if outer.Scope.Symbol != "outer" || outer.Rules != 1 || len(outer.Nested) != 1 {
		t.Fatalf("outer keeps its nested function under the cyclomatic filter: %+v", outer)
	}
	if outer.Findings[0].RuleID != domain.RuleID(domain.MetricCyclomatic) {
		t.Fatalf("only the requested rule remains: %+v", outer.Findings)
	}
}

func TestWorklistRefusesUnknownRulesAndReportsIncompleteAnalysis(t *testing.T) {
	unknown := execute(t, "", "worklist", "--root", worklistProject(t), "--rule", "nope")
	if unknown.code != 2 || !strings.Contains(unknown.stderr, "unknown rule") {
		t.Fatalf("an unknown rule must be refused: %+v", unknown)
	}
	broken := execute(t, "", "worklist", "--root", project(t, "export function broken( {\n", blockingCyclomatic()))
	if broken.code != 2 || !strings.Contains(broken.stdout, "incomplete") {
		t.Fatalf("an incomplete analysis must exit 2 and say so: %+v", broken)
	}
}
