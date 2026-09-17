package typescript

import (
	"slices"
	"strings"
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

func analyze(t *testing.T, name, code string) *adapter.SourceFile {
	t.Helper()
	return Analyze(name, "/repo/"+name, code)
}

func value(t *testing.T, file *adapter.SourceFile, id domain.MetricID, symbol string) float64 {
	t.Helper()
	for _, measurement := range file.Measurements {
		if measurement.MetricID == id && measurement.Scope.Symbol == symbol {
			if measurement.Value == nil {
				t.Fatalf("%s for %s has no value", id, symbol)
			}
			return *measurement.Value
		}
	}
	t.Fatalf("no %s measurement for symbol %q; symbols: %v", id, symbol, symbols(file))
	return 0
}

func symbols(file *adapter.SourceFile) []string {
	seen := map[string]bool{}
	var result []string
	for _, measurement := range file.Measurements {
		if !seen[measurement.Scope.Symbol] {
			seen[measurement.Scope.Symbol] = true
			result = append(result, measurement.Scope.Symbol)
		}
	}
	return result
}

func expect(t *testing.T, file *adapter.SourceFile, id domain.MetricID, symbol string, want float64) {
	t.Helper()
	if got := value(t, file, id, symbol); got != want {
		t.Errorf("%s(%s) = %v, want %v", id, symbol, got, want)
	}
}

func TestReferenceFixtureCyclomaticNestingParametersCognitive(t *testing.T) {
	file := analyze(t, "sample.ts", `function score(a: boolean, b: boolean, c: boolean) {
  if (a && b) {
    for (let i = 0; i < 2; i++) {
      if (c) return i;
    }
  }
  return 0;
}
`)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", file.Diagnostics)
	}
	expect(t, file, domain.MetricCyclomatic, "score", 5)
	expect(t, file, domain.MetricNestingDepth, "score", 3)
	expect(t, file, domain.MetricParameterCount, "score", 3)
	expect(t, file, domain.MetricCognitive, "score", 7)
	expect(t, file, domain.MetricFunctionLines, "score", 8)
}

func TestNestedFunctionsResetCyclomaticAndNesting(t *testing.T) {
	file := analyze(t, "sample.ts", `function outer() {
  const inner = (ready: boolean) => ready ? 1 : 0;
  return inner(true);
}
`)
	expect(t, file, domain.MetricCyclomatic, "outer", 1)
	expect(t, file, domain.MetricCyclomatic, "outer.inner", 2)
	expect(t, file, domain.MetricNestingDepth, "outer", 0)
	expect(t, file, domain.MetricNestingDepth, "outer.inner", 1)
	// Cognitive complexity attributes the nested ternary (nesting level 1) to outer as well.
	expect(t, file, domain.MetricCognitive, "outer", 2)
	expect(t, file, domain.MetricCognitive, "outer.inner", 1)
}

func TestFunctionLinesExcludeCommentOnlyAndBlankLines(t *testing.T) {
	file := analyze(t, "sample.ts", `function score() {
  // explanation

  const value = 1;
  /* another
     comment */
  return value;
}
`)
	expect(t, file, domain.MetricFunctionLines, "score", 4)
}

func TestParsesJSXAndTSX(t *testing.T) {
	file := analyze(t, "card.tsx", `type Props = { ready: boolean };
export const Card = ({ ready }: Props) => <section>{ready ? "yes" : "no"}</section>;
`)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", file.Diagnostics)
	}
	expect(t, file, domain.MetricCyclomatic, "Card", 2)
	expect(t, file, domain.MetricParameterCount, "Card", 1)

	jsx := analyze(t, "card.jsx", `export function Card({ ready }) { return <div className="x">{ready && <b>on</b>}</div>; }`)
	if len(jsx.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", jsx.Diagnostics)
	}
	expect(t, jsx, domain.MetricCyclomatic, "Card", 2)
}

func TestSyntaxErrorIsRequiredDiagnosticNotCleanResult(t *testing.T) {
	file := analyze(t, "broken.ts", "export function broken( {\n")
	if len(file.Measurements) != 0 {
		t.Fatalf("expected no measurements, got %d", len(file.Measurements))
	}
	if len(file.Diagnostics) != 1 || file.Diagnostics[0].Code != "typescript.parse_failed" || !file.Diagnostics[0].IsRequiredError() {
		t.Fatalf("expected a required parse diagnostic, got %+v", file.Diagnostics)
	}
	if !strings.Contains(file.Diagnostics[0].Message, "Could not parse source") {
		t.Fatalf("unexpected message %q", file.Diagnostics[0].Message)
	}
}

func TestCognitiveComplexitySpecificationFixtures(t *testing.T) {
	cases := []struct {
		name   string
		symbol string
		code   string
		want   float64
	}{
		{"labelled jump example", "sumOfPrimes", `function sumOfPrimes(max: number) {
  let total = 0;
  OUT: for (let i = 1; i <= max; ++i) {
    for (let j = 2; j < i; ++j) {
      if (i % j == 0) {
        continue OUT;
      }
    }
    total += i;
  }
  return total;
}`, 7},
		{"switch counts once", "getWords", `function getWords(number: number) {
  switch (number) {
    case 1: return "one";
    case 2: return "a couple";
    default: return "lots";
  }
}`, 1},
		{"nesting example", "myMethod", `function myMethod() {
  try {
    if (condition1) {
      for (let i = 0; i < 10; i++) {
        while (condition2) { }
      }
    }
  } catch (someError) {
  }
}`, 7},
		{"lambda raises nesting", "myMethod2", `function myMethod2() {
  const r = () => {
    if (condition1) {
    }
  };
}`, 2},
		{"else if chain is flat", "chain", `function chain(a: boolean, b: boolean) {
  if (a) {
  } else if (b) {
  } else {
  }
}`, 3},
		{"logical operator sequences", "logic", `function logic(a: boolean, b: boolean, c: boolean, d: boolean, e: boolean, f: boolean) {
  if (a && b && c || d || e && f) {
  }
}`, 4},
		{"parentheses do not restart same operator", "parens", `function parens(a: boolean, b: boolean, c: boolean) {
  return a && (b && c);
}`, 1},
		{"ternary nested in if", "nestedTernary", `function nestedTernary(a: boolean, b: boolean) {
  if (a) {
    return b ? 1 : 2;
  }
  return 0;
}`, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := analyze(t, "spec.ts", tc.code)
			if len(file.Diagnostics) != 0 {
				t.Fatalf("unexpected diagnostics: %+v", file.Diagnostics)
			}
			expect(t, file, domain.MetricCognitive, tc.symbol, tc.want)
		})
	}
}

func TestSymbolNamingCoversDeclarationContexts(t *testing.T) {
	file := analyze(t, "names.ts", `export class Service {
  constructor(private readonly x: number) {}
  run() { return () => 1; }
  get size() { return 1; }
  static create = () => new Service(1);
}
const handler = function named() {};
const { a } = () => ({ a: 1 });
const obj = { key: () => 1, method() {}, "quoted": function () {} };
export default function () {}
function dup() {}
function dup() {}
`)
	want := []string{
		"Service.constructor",
		"Service.run",
		"Service.run.<anonymous>",
		"Service.size",
		"Service.create",
		"handler",
		"<destructured>",
		"key",
		"method",
		"quoted",
		"<anonymous>",
		"dup",
		"dup#2",
	}
	got := symbols(file)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("symbols = %v, want %v", got, want)
	}
}

func TestImportsAreCollectedByKind(t *testing.T) {
	file := analyze(t, "imports.ts", `import { a } from "./a";
import type { T } from "./types";
export * from "./all";
export { b } from "./b";
import legacy = require("./legacy");
const lazy = () => import("./lazy");
const common = require("./common");
const computed = require(name);
`)
	want := []adapter.Import{
		{Specifier: "./a", Line: 1, Kind: adapter.ImportStatic},
		{Specifier: "./types", Line: 2, Kind: adapter.ImportStatic},
		{Specifier: "./all", Line: 3, Kind: adapter.ImportStatic},
		{Specifier: "./b", Line: 4, Kind: adapter.ImportStatic},
		{Specifier: "./legacy", Line: 5, Kind: adapter.ImportRequire},
		{Specifier: "./lazy", Line: 6, Kind: adapter.ImportDynamic},
		{Specifier: "./common", Line: 7, Kind: adapter.ImportRequire},
	}
	if len(file.Imports) != len(want) {
		t.Fatalf("imports = %+v, want %+v", file.Imports, want)
	}
	for i := range want {
		if file.Imports[i] != want[i] {
			t.Errorf("imports[%d] = %+v, want %+v", i, file.Imports[i], want[i])
		}
	}
}

func TestTokensTreatContextDependentConstructsAsSingleTokens(t *testing.T) {
	file := analyze(t, "tokens.tsx", "const re = /a\\/b/g; // trailing\n"+
		"const tpl = `x ${re} y ${1} z`;\n"+
		"const el = <p>Don't /stop/ {tpl}</p>;\n")
	if len(file.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", file.Diagnostics)
	}
	var values []string
	for _, token := range file.Tokens {
		values = append(values, token.Value)
	}
	joined := strings.Join(values, " ")
	for _, expected := range []string{"/a\\/b/g", "`x ${", "} y ${", "} z`", "Don't /stop/"} {
		if !strings.Contains(joined, expected) {
			t.Errorf("token stream %q lacks %q", joined, expected)
		}
	}
	if strings.Contains(joined, "trailing") {
		t.Errorf("comments must not be tokens: %q", joined)
	}
	for _, token := range file.Tokens {
		if token.Index < 0 || token.Line < 1 || token.EndLine < token.Line {
			t.Fatalf("invalid token positions: %+v", token)
		}
	}
}

func TestAdapterIdentity(t *testing.T) {
	a := New()
	if a.ID() != ID || a.Version() != Version {
		t.Fatalf("unexpected identity %s %s", a.ID(), a.Version())
	}
	for _, name := range []string{"a.ts", "b.tsx", "c.js", "d.jsx", "e.mjs", "f.cjs"} {
		if !a.Supports(name) {
			t.Errorf("expected %s to be supported", name)
		}
	}
	if a.Supports("x.py") || a.Supports("y.java") {
		t.Error("unexpected language support")
	}
	if a.Analyzers()[ParserName] != ParserVersion {
		t.Error("parser version must be declared")
	}
}

func TestImportSpansCoverImportsAndReExports(t *testing.T) {
	code := "import {\n  alpha,\n  beta,\n} from \"./ab\";\nexport { gamma } from \"./gamma\";\nexport { alpha };\nimport fs = require(\"fs\");\nconst delta = require(\"./delta\");\nexport const use = () => [alpha, beta, fs, delta];\n"
	spans := Analyze("spans.ts", "/repo/spans.ts", code).ImportSpans
	if !slices.Equal(spans, []adapter.LineSpan{{Line: 1, EndLine: 4}, {Line: 5, EndLine: 5}, {Line: 7, EndLine: 7}}) {
		t.Fatalf("imports, re-exports and import-equals are spans; a plain export and a require call are not: %v", spans)
	}
}
