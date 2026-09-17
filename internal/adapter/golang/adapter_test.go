package golang

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
)

func analyze(t *testing.T, name, code string) *adapter.SourceFile {
	t.Helper()
	file := Analyze(name, "/repo/"+name, code, adapter.Module{})
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
	file := analyze(t, "sample.go", `package sample

func score(a, b bool, c int) int {
	if a && b {
		for i := 0; i < 2; i++ {
			if c > 0 {
				return i
			}
		}
	}
	return 0
}
`)
	expect(t, file, domain.MetricCyclomatic, "score", 5)
	expect(t, file, domain.MetricNestingDepth, "score", 3)
	expect(t, file, domain.MetricParameterCount, "score", 3)
	expect(t, file, domain.MetricCognitive, "score", 7)
	expect(t, file, domain.MetricFunctionLines, "score", 10)
}

func TestMethodsLiteralsAndNaming(t *testing.T) {
	file := analyze(t, "names.go", `package sample

type Store[K comparable, V any] struct{}

func (s *Store[K, V]) Get(k K) V {
	load := func(ready bool) int {
		if ready {
			return 1
		}
		return 0
	}
	go func() { load(true) }()
	var zero V
	return zero
}

func (Store[K, V]) Len() int { return 0 }

var handler = func(a, b int) int { return a + b }

func dup() {}
func dup() {}
`)
	want := []string{"Store.Get", "Store.Get.load", "Store.Get.<anonymous>", "Store.Len", "handler", "dup", "dup#2"}
	if got := symbols(file); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("symbols = %v, want %v", got, want)
	}
	// The receiver is not a parameter; the literal's if counts in the method's cognitive total.
	expect(t, file, domain.MetricParameterCount, "Store.Get", 1)
	expect(t, file, domain.MetricParameterCount, "handler", 2)
	expect(t, file, domain.MetricCyclomatic, "Store.Get", 1)
	expect(t, file, domain.MetricCyclomatic, "Store.Get.load", 2)
	expect(t, file, domain.MetricCognitive, "Store.Get", 2)
	expect(t, file, domain.MetricCognitive, "Store.Get.load", 1)
	expect(t, file, domain.MetricNestingDepth, "Store.Get", 0)
}

func TestSwitchSelectElseChainsAndJumps(t *testing.T) {
	file := analyze(t, "branches.go", `package sample

func branches(a, b int, v any, ch chan int) string {
	if a > 0 {
		return "a"
	} else if b > 0 {
		return "b"
	} else {
		if v == nil {
			return "nil"
		}
	}
	switch a {
	case 1, 2:
		return "small"
	default:
	}
	switch v.(type) {
	case int:
	}
	select {
	case <-ch:
	default:
	}
outer:
	for i := 0; i < a; i++ {
		for range b {
			if i == b {
				continue outer
			}
			break
		}
	}
	return ""
}
`)
	// 1 + if, else-if, nested if, one clause (case 1, 2 counts once), type case, comm clause, 2 loops, nested if = 10
	expect(t, file, domain.MetricCyclomatic, "branches", 10)
	// if 1, else-if 1, else 1, nested if 2 (nesting 1), switch 1, type switch 1, select 1,
	// for 1, range 2, if 3, labelled continue 1 = 15
	expect(t, file, domain.MetricCognitive, "branches", 15)
	expect(t, file, domain.MetricNestingDepth, "branches", 3)
}

func TestLogicalSequencesLookThroughParentheses(t *testing.T) {
	file := analyze(t, "logic.go", `package sample

func ok(a, b, c, d bool) bool {
	return a && b && (c || d)
}
`)
	// 1 + three operators
	expect(t, file, domain.MetricCyclomatic, "ok", 4)
	// one && run and one || run
	expect(t, file, domain.MetricCognitive, "ok", 2)
}

func TestFunctionLinesExcludeCommentsBlankLinesAndDocComments(t *testing.T) {
	file := analyze(t, "lines.go", `package sample

// score is documented above the func keyword, so the doc comment is outside it.
func score() int {
	// explanation

	value := 1 /* inline */
	raw := `+"`multi\nline`"+`
	_ = raw
	return value // trailing
}
`)
	// The raw string's second line is source text, so it counts.
	expect(t, file, domain.MetricFunctionLines, "score", 7)
}

func TestParameterSlots(t *testing.T) {
	file := analyze(t, "params.go", `package sample

func grouped(a, b int, c string, rest ...int) {}
func unnamed(int, string) {}
func none() {}
`)
	expect(t, file, domain.MetricParameterCount, "grouped", 4)
	expect(t, file, domain.MetricParameterCount, "unnamed", 2)
	expect(t, file, domain.MetricParameterCount, "none", 0)
}

func TestImportsAreCollected(t *testing.T) {
	file := analyze(t, "imports.go", `package sample

import (
	"fmt"
	str "strings"
	_ "embed"
	. "math"

	"example.com/app/internal/domain"
)

var _ = fmt.Sprint(str.ToUpper(""), Pi, domain.X)
`)
	want := []adapter.Import{
		{Specifier: "fmt", Line: 4, Kind: adapter.ImportStatic},
		{Specifier: "strings", Line: 5, Kind: adapter.ImportStatic},
		{Specifier: "embed", Line: 6, Kind: adapter.ImportStatic},
		{Specifier: "math", Line: 7, Kind: adapter.ImportStatic},
		{Specifier: "example.com/app/internal/domain", Line: 9, Kind: adapter.ImportStatic},
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
	file := Analyze("broken.go", "/repo/broken.go", "package sample\n\nfunc broken( {\n", adapter.Module{})
	if len(file.Measurements) != 0 || len(file.Diagnostics) != 1 || file.Diagnostics[0].Code != "go.parse_failed" || !file.Diagnostics[0].IsRequiredError() {
		t.Fatalf("expected a required parse diagnostic, got %+v / %+v", file.Measurements, file.Diagnostics)
	}
}

func TestTokensExcludeCommentsAndNormaliseSemicolons(t *testing.T) {
	file := analyze(t, "tokens.go", "package sample // comment\n\nvar x = 1; var y = `a\nb`\n")
	var kinds, values []string
	for _, token := range file.Tokens {
		kinds = append(kinds, token.Type)
		values = append(values, token.Value)
	}
	wantKinds := "package IDENT ; var IDENT = INT ; var IDENT = STRING ;"
	if got := strings.Join(kinds, " "); got != wantKinds {
		t.Fatalf("token kinds = %q, want %q", got, wantKinds)
	}
	if values[2] != ";" || values[7] != ";" || values[len(values)-1] != ";" {
		t.Fatalf("semicolons must be normalised, got %q", values)
	}
	if last := file.Tokens[len(file.Tokens)-2]; last.Line != 3 || last.EndLine != 4 {
		t.Fatalf("raw string must span its lines, got %+v", last)
	}
}

func TestAnalyzeFileFindsTheNearestModule(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/app // comment\n\ngo 1.27\n")
	write("internal/app/service.go", "package app\n")
	write("tools/gen/go.mod", "module \"example.com/app/tools/gen\"\n")
	write("tools/gen/main.go", "package main\n")
	write("plain.go", "package plain\n")

	a := New()
	cases := map[string]adapter.Module{
		"internal/app/service.go": {Package: "example.com/app/internal/app", Root: "example.com/app"},
		"tools/gen/main.go":       {Package: "example.com/app/tools/gen", Root: "example.com/app/tools/gen"},
		"plain.go":                {Package: "example.com/app", Root: "example.com/app"},
	}
	for name, want := range cases {
		file, err := a.AnalyzeFile(context.Background(), root, filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if file.Module != want {
			t.Errorf("%s: module = %+v, want %+v", name, file.Module, want)
		}
	}

	// A root without go.mod gives no module, so no import can be internal.
	bare := t.TempDir()
	if err := os.WriteFile(filepath.Join(bare, "x.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := a.AnalyzeFile(context.Background(), bare, filepath.Join(bare, "x.go"))
	if err != nil || file.Module != (adapter.Module{}) {
		t.Fatalf("expected no module, got %+v %v", file.Module, err)
	}
}

func TestAdapterIdentity(t *testing.T) {
	a := New()
	if a.ID() != ID || !a.Supports("x.go") || a.Supports("x.ts") || a.Resolver() == nil {
		t.Fatalf("unexpected adapter identity")
	}
}

func TestImportSpansCoverTheImportBlock(t *testing.T) {
	code := "package sample\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nimport \"strings\"\n\nfunc use() { fmt.Println(os.Args, strings.ToUpper(\"x\")) }\n"
	spans := Analyze("spans.go", "/repo/spans.go", code, adapter.Module{}).ImportSpans
	if !slices.Equal(spans, []adapter.LineSpan{{Line: 3, EndLine: 6}, {Line: 8, EndLine: 8}}) {
		t.Fatalf("the import block and the single import must each be a span, got %v", spans)
	}
}
