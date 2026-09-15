package python

import (
	"strings"
	"testing"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
)

func analyze(t *testing.T, name, code string) *adapter.SourceFile {
	t.Helper()
	file := Analyze(name, "/repo/"+name, code)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", file.Diagnostics)
	}
	return file
}

func value(t *testing.T, file *adapter.SourceFile, id domain.MetricID, symbol string) float64 {
	t.Helper()
	for _, measurement := range file.Measurements {
		if measurement.MetricID == id && measurement.Scope.Symbol == symbol {
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

func TestReferenceFixture(t *testing.T) {
	file := analyze(t, "sample.py", `def score(a, b, c):
    if a and b:
        for i in range(2):
            if c:
                return i
    return 0
`)
	expect(t, file, domain.MetricCyclomatic, "score", 5)
	expect(t, file, domain.MetricNestingDepth, "score", 3)
	expect(t, file, domain.MetricParameterCount, "score", 3)
	expect(t, file, domain.MetricCognitive, "score", 7)
	expect(t, file, domain.MetricFunctionLines, "score", 6)
}

func TestNestedFunctionsAndLambdas(t *testing.T) {
	file := analyze(t, "sample.py", `def outer():
    inner = lambda ready: 1 if ready else 0
    def helper(x):
        return x
    return inner(True)
`)
	expect(t, file, domain.MetricCyclomatic, "outer", 1)
	expect(t, file, domain.MetricCyclomatic, "outer.<lambda>", 2)
	expect(t, file, domain.MetricCyclomatic, "outer.helper", 1)
	expect(t, file, domain.MetricCognitive, "outer", 2)
	expect(t, file, domain.MetricCognitive, "outer.<lambda>", 1)
	expect(t, file, domain.MetricNestingDepth, "outer", 0)
	expect(t, file, domain.MetricParameterCount, "outer.<lambda>", 1)
}

func TestFunctionLinesExcludeCommentsBlankLinesAndDecorators(t *testing.T) {
	file := analyze(t, "sample.py", `@decorator
def score():
    # explanation

    value = 1
    """not a comment but a string"""
    return value  # trailing
`)
	expect(t, file, domain.MetricFunctionLines, "score", 4)
}

func TestElifElseComprehensionsExceptAndMatch(t *testing.T) {
	file := analyze(t, "sample.py", `def branches(a, b, items, point):
    if a:
        pass
    elif b:
        pass
    else:
        pass
    squares = [x * x for x in items if x > 0 if x < 10]
    try:
        run()
    except ValueError:
        pass
    except (TypeError, KeyError) as error:
        raise
    match point:
        case (0, 0):
            return "origin"
        case _:
            return "other"
`)
	// cyclomatic: if, elif, comprehension for + 2 ifs, 2 handlers, 1 non-wildcard case = 1 + 8
	expect(t, file, domain.MetricCyclomatic, "branches", 9)
	// cognitive: if +1, elif +1, else +1, except +1, except +1, match +1 = 6
	expect(t, file, domain.MetricCognitive, "branches", 6)
	// nesting: elif is nested in if (2); handlers 1; match 1
	expect(t, file, domain.MetricNestingDepth, "branches", 2)
}

func TestElseContainingIfIsNotAnElif(t *testing.T) {
	file := analyze(t, "sample.py", `def f(a, b):
    if a:
        pass
    else:
        if b:
            pass
`)
	// if +1, else +1, nested if +1 + nesting 1 = 4
	expect(t, file, domain.MetricCognitive, "f", 4)
}

func TestSymbolNamingCoversClassesNestingAndDuplicates(t *testing.T) {
	file := analyze(t, "names.py", `class Service:
    def __init__(self, x):
        self.x = x

    def run(self):
        return lambda: 1

    class Inner:
        def go(self): pass

def dup(): pass
def dup(): pass
async def fetch(): pass
`)
	want := []string{"Service.__init__", "Service.run", "Service.run.<lambda>", "Inner.go", "dup", "dup#2", "fetch"}
	if got := symbols(file); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("symbols = %v, want %v", got, want)
	}
	expect(t, file, domain.MetricParameterCount, "Service.__init__", 2)
}

func TestImportsAreCollected(t *testing.T) {
	file := analyze(t, "imports.py", `import os.path as osp, sys
from . import sibling
from ..pkg.module import a, b as c
from pkg import *
import importlib
mod = importlib.import_module("dynamic.module")
other = __import__("legacy")
computed = importlib.import_module(name)
`)
	want := []adapter.Import{
		{Specifier: "os.path", Line: 1, Kind: adapter.ImportStatic},
		{Specifier: "sys", Line: 1, Kind: adapter.ImportStatic},
		{Specifier: ".sibling", Line: 2, Kind: adapter.ImportStatic},
		{Specifier: "..pkg.module.a", Line: 3, Kind: adapter.ImportStatic},
		{Specifier: "..pkg.module.b", Line: 3, Kind: adapter.ImportStatic},
		{Specifier: "pkg.*", Line: 4, Kind: adapter.ImportStatic},
		{Specifier: "importlib", Line: 5, Kind: adapter.ImportStatic},
		{Specifier: "dynamic.module", Line: 6, Kind: adapter.ImportDynamic},
		{Specifier: "legacy", Line: 7, Kind: adapter.ImportDynamic},
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

func TestSyntaxErrorIsRequiredDiagnostic(t *testing.T) {
	file := Analyze("broken.py", "/repo/broken.py", "def broken(:\n    pass\n")
	if len(file.Measurements) != 0 || len(file.Diagnostics) != 1 || file.Diagnostics[0].Code != "python.parse_failed" || !file.Diagnostics[0].IsRequiredError() {
		t.Fatalf("expected a required parse diagnostic, got %+v / %+v", file.Measurements, file.Diagnostics)
	}
}

func TestTokensExcludeCommentsAndKeepStructure(t *testing.T) {
	file := analyze(t, "tokens.py", "x = 1  # comment\nif x:\n    y = f\"{x}\"\n")
	var kinds []string
	for _, token := range file.Tokens {
		kinds = append(kinds, token.Type)
	}
	want := "Name Op Number Newline Name Name Op Newline Indent Name Op String Newline Dedent"
	if got := strings.Join(kinds, " "); got != want {
		t.Fatalf("token kinds = %q, want %q", got, want)
	}
}

func TestAdapterIdentity(t *testing.T) {
	a := New()
	if a.ID() != ID || !a.Supports("x.py") || a.Supports("x.ts") || a.Resolver() == nil {
		t.Fatalf("unexpected adapter identity")
	}
}
