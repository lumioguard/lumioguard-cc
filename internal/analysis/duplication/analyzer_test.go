package duplication

import (
	"context"
	"strings"
	"testing"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/typescript"
	"github.com/lumiostack/lumioguard-cc/internal/analysis"
	"github.com/lumiostack/lumioguard-cc/internal/config"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
)

const repeated = `export function calculate(value: number) {
  const adjusted = value + 10;
  return adjusted * 2;
}`

func TestExactTokenClonesProduceDensityAndFindings(t *testing.T) {
	cfg := config.Default()
	cfg.Metrics.DuplicationPercent.MinTokens = 8
	cfg.Metrics.DuplicationPercent.MinLines = 2
	files := []*adapter.SourceFile{
		typescript.Analyze("a.ts", "/r/a.ts", repeated),
		typescript.Analyze("b.ts", "/r/b.ts", strings.Replace(repeated, "calculate", "calculateAgain", 1)),
	}
	result, err := New("recheck").Analyze(context.Background(), analysis.Input{Files: files, Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	density := result.Measurements[0]
	if density.MetricID != domain.MetricTokenCloneDensity || density.Value == nil || *density.Value <= 0 {
		t.Fatalf("expected positive density, got %+v", density)
	}
	if density.Evidence["cloneGroups"].(int) < 1 {
		t.Fatalf("expected clone groups in evidence, got %+v", density.Evidence)
	}
	if len(result.Findings) == 0 || result.Findings[0].RuleID != domain.RuleTokenClone {
		t.Fatalf("expected clone findings, got %+v", result.Findings)
	}
}

func TestNoSourcesIsNotApplicable(t *testing.T) {
	result, err := New("recheck").Analyze(context.Background(), analysis.Input{Config: config.Default()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Measurements[0].Status != domain.StatusNotApplicable || result.Measurements[0].Value != nil {
		t.Fatalf("expected not_applicable, got %+v", result.Measurements[0])
	}
}

func TestFindingsRequireThresholdBreach(t *testing.T) {
	cfg := config.Default()
	cfg.Metrics.DuplicationPercent.MinTokens = 8
	cfg.Metrics.DuplicationPercent.MinLines = 2
	cfg.Metrics.DuplicationPercent.Threshold = 100
	files := []*adapter.SourceFile{
		typescript.Analyze("a.ts", "/r/a.ts", repeated),
		typescript.Analyze("b.ts", "/r/b.ts", repeated),
	}
	result, err := New("recheck").Analyze(context.Background(), analysis.Input{Files: files, Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("no findings expected below threshold, got %+v", result.Findings)
	}
}

const copiedBody = `(items: number[]) {
  let total = 0;
  for (const item of items) {
    if (item > 10) {
      total += item * 2;
    } else {
      total += item;
    }
  }
  return total;
}`

func analyze(t *testing.T, minTokens, minLines int, sources map[string]string) analysis.Result {
	t.Helper()
	cfg := config.Default()
	cfg.Metrics.DuplicationPercent.MinTokens = minTokens
	cfg.Metrics.DuplicationPercent.MinLines = minLines
	var files []*adapter.SourceFile
	for _, name := range []string{"a.ts", "b.ts", "c.ts"} {
		if code, ok := sources[name]; ok {
			files = append(files, typescript.Analyze(name, "/r/"+name, code))
		}
	}
	result, err := New("recheck").Analyze(context.Background(), analysis.Input{Files: files, Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// A block copied whole is one region, not one finding per token window.
func TestACopiedBlockIsOneRegionNamedAfterItsFunctions(t *testing.T) {
	result := analyze(t, 8, 2, map[string]string{
		"a.ts": "export function alpha" + copiedBody,
		"b.ts": "export function beta" + copiedBody,
	})
	if len(result.Findings) != 1 {
		t.Fatalf("expected one finding for one copied block, got %d: %+v", len(result.Findings), result.Findings)
	}
	finding := result.Findings[0]
	if finding.Scope.Key != "clone:a.ts::alpha|b.ts::beta" {
		t.Fatalf("identity must name the files and functions, got %q", finding.Scope.Key)
	}
	occurrences := finding.Evidence["occurrences"].([]occurrence)
	if len(occurrences) != 2 || occurrences[0].StartLine != 1 || occurrences[0].EndLine != 11 || occurrences[1].Symbol != "beta" {
		t.Fatalf("occurrences must cover the whole block: %+v", occurrences)
	}
	if finding.Current == nil || *finding.Current != 22 || finding.Unit != domain.UnitLines {
		t.Fatalf("current must be the duplicated lines of every copy, got %+v", finding)
	}
	if result.Measurements[0].Evidence["cloneGroups"].(int) != 1 {
		t.Fatalf("expected one clone group, got %+v", result.Measurements[0].Evidence)
	}
}

// Files that import the same modules are not copies of each other.
func TestImportDeclarationsAreNeverClones(t *testing.T) {
	imports := "import { x } from \"./x\";\nimport { y } from \"./y\";\nimport { z } from \"./z\";\n"
	result := analyze(t, 8, 2, map[string]string{
		"a.ts": imports + "export const a = x + y + z + 1;\n",
		"b.ts": imports + "export const b = x * y * z * 2;\n",
	})
	if len(result.Findings) != 0 || *result.Measurements[0].Value != 0 {
		t.Fatalf("shared imports must not count as duplication: %+v %+v", result.Findings, result.Measurements[0])
	}
}

// Code on both sides of an import declaration is not one window.
func TestAWindowNeverCrossesAnImportDeclaration(t *testing.T) {
	result := analyze(t, 10, 2, map[string]string{
		"a.ts": "const p = 1;\nimport { x } from \"./x\";\nconst q = p + x;\n",
		"b.ts": "const p = 1;\nconst q = p + x;\n",
	})
	if len(result.Findings) != 0 || *result.Measurements[0].Value != 0 {
		t.Fatalf("tokens around an import must not join into one window: %+v", result.Findings)
	}
}

// Two different copies between the same functions keep distinct identities.
func TestDistinctRegionsBetweenTheSameFunctionsGetOrdinals(t *testing.T) {
	first := "  const a = [1, 2, 3, 4, 5, 6, 7, 8, 9];\n  const b = a.map((v) => v * 2);\n"
	second := "  const c = { one: 1, two: 2, three: 3, four: 4 };\n  const d = Object.keys(c).length;\n"
	result := analyze(t, 8, 2, map[string]string{
		"a.ts": "export function alpha() {\n" + first + "  alpha();\n" + second + "  return b.length + d;\n}\n",
		"b.ts": "export function beta() {\n" + first + "  beta();\n" + second + "  return d - b.length;\n}\n",
	})
	if len(result.Findings) != 2 {
		t.Fatalf("expected two regions, got %d: %+v", len(result.Findings), result.Findings)
	}
	keys := []string{result.Findings[0].Scope.Key, result.Findings[1].Scope.Key}
	if keys[0] != "clone:a.ts::alpha|b.ts::beta" || keys[1] != "clone:a.ts::alpha|b.ts::beta#2" {
		t.Fatalf("expected ordinal keys, got %v", keys)
	}
}

// The order of unrelated files does not change which regions are found.
func TestRegionsDoNotDependOnFileOrder(t *testing.T) {
	sources := map[string]string{
		"a.ts": "export function alpha" + copiedBody,
		"b.ts": "export function beta" + copiedBody,
		"c.ts": "export function gamma" + strings.Replace(copiedBody, "total += item;", "total += item + 1;", 1),
	}
	forward := analyze(t, 8, 2, sources)
	cfg := config.Default()
	cfg.Metrics.DuplicationPercent.MinTokens = 8
	cfg.Metrics.DuplicationPercent.MinLines = 2
	var reversed []*adapter.SourceFile
	for _, name := range []string{"c.ts", "b.ts", "a.ts"} {
		reversed = append(reversed, typescript.Analyze(name, "/r/"+name, sources[name]))
	}
	backward, err := New("recheck").Analyze(context.Background(), analysis.Input{Files: reversed, Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if len(forward.Findings) != len(backward.Findings) {
		t.Fatalf("file order changed the regions: %d vs %d", len(forward.Findings), len(backward.Findings))
	}
	for i := range forward.Findings {
		if forward.Findings[i].Scope.Key != backward.Findings[i].Scope.Key || *forward.Findings[i].Current != *backward.Findings[i].Current {
			t.Fatalf("file order changed region %d: %+v vs %+v", i, forward.Findings[i], backward.Findings[i])
		}
	}
}
