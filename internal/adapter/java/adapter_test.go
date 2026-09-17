package java

import (
	"slices"
	"strings"
	"testing"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
)

const fixture = `package com.acme;

import java.util.List;
import com.acme.util.Helper;
import com.acme.util.*;
import static com.acme.util.Strings.trim;

/** Service documentation. */
public class Service {
    private final List<String> items;

    public Service(List<String> items) { this.items = items; }

    @Override
    public int score(boolean a, boolean b, boolean c) {
        // comment line
        if (a && b) {
            for (int i = 0; i < 2; i++) {
                if (c) return i;
            }
        }
        return 0;
    }

    public Runnable outer() {
        Runnable inner = () -> { int x = items.isEmpty() ? 1 : 0; };
        return inner;
    }

    public String describe(int n) {
        switch (n) {
            case 1: return "one";
            case 2:
            case 3: return "few";
            default: return "many";
        }
    }

    public String modern(Object o) {
        String block = """
            text block
            """;
        return switch (o) {
            case Integer i when i > 0 -> "positive";
            case String s -> s + block;
            default -> "other";
        };
    }

    public void loops(Other other) {
        OUTER:
        for (String item : items) {
            try {
                if (item.isEmpty()) continue OUTER;
            } catch (RuntimeException | Error e) {
                throw e;
            } finally {
                cleanup();
            }
        }
        do { Helper.help(); } while (items.isEmpty());
    }

    private void cleanup() {}

    public static class Inner {
        void run() { new Thread(new Runnable() { public void run() {} }).start(); }
    }
}
`

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

func TestReferenceFixtureMetrics(t *testing.T) {
	file := analyze(t, "src/main/java/com/acme/Service.java", fixture)
	expect(t, file, domain.MetricCyclomatic, "Service.score", 5)
	expect(t, file, domain.MetricNestingDepth, "Service.score", 3)
	expect(t, file, domain.MetricParameterCount, "Service.score", 3)
	expect(t, file, domain.MetricCognitive, "Service.score", 7)
	// @Override, signature, if, for, if, }, }, return, } = 9 lines; the comment line is excluded
	expect(t, file, domain.MetricFunctionLines, "Service.score", 9)

	expect(t, file, domain.MetricParameterCount, "Service.<init>", 1)
	expect(t, file, domain.MetricCyclomatic, "Service.outer", 1)
	expect(t, file, domain.MetricCyclomatic, "Service.outer.<lambda>", 2)
	expect(t, file, domain.MetricCognitive, "Service.outer", 2)

	expect(t, file, domain.MetricCyclomatic, "Service.describe", 4)
	expect(t, file, domain.MetricCognitive, "Service.describe", 1)
	expect(t, file, domain.MetricCyclomatic, "Service.modern", 3)
	expect(t, file, domain.MetricCognitive, "Service.modern", 1)

	// for +1, if +2, labelled continue +1, catch +2, do-while +1
	expect(t, file, domain.MetricCyclomatic, "Service.loops", 5)
	expect(t, file, domain.MetricCognitive, "Service.loops", 7)
	expect(t, file, domain.MetricNestingDepth, "Service.loops", 2)
}

func TestSymbolNaming(t *testing.T) {
	file := analyze(t, "Service.java", fixture)
	want := []string{
		"Service.<init>", "Service.score", "Service.outer", "Service.outer.<lambda>", "Service.describe",
		"Service.modern", "Service.loops", "Service.cleanup", "Inner.run", "<anonymous-class>.run.run",
	}
	if got := symbols(file); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("symbols = %v, want %v", got, want)
	}
}

func TestModuleImportsAndImplicitReferences(t *testing.T) {
	file := analyze(t, "src/main/java/com/acme/Service.java", fixture)
	if file.Module.Package != "com.acme" {
		t.Fatalf("unexpected module %+v", file.Module)
	}
	var declared, implicit []string
	for _, imp := range file.Imports {
		if imp.Kind == adapter.ImportImplicit {
			implicit = append(implicit, imp.Specifier)
		} else {
			declared = append(declared, imp.Specifier)
		}
	}
	wantDeclared := "java.util.List,com.acme.util.Helper,com.acme.util.*,com.acme.util.Strings.trim"
	if got := strings.Join(declared, ","); got != wantDeclared {
		t.Fatalf("declared imports = %q, want %q", got, wantDeclared)
	}
	for _, name := range []string{"String", "Runnable", "Thread", "Integer", "Object", "Other", "RuntimeException", "Error", "Override"} {
		if !contains(implicit, name) {
			t.Errorf("implicit references %v lack %s", implicit, name)
		}
	}
	for _, name := range []string{"List", "Helper", "Service", "Inner", "items"} {
		if contains(implicit, name) {
			t.Errorf("implicit references must not contain %s: %v", name, implicit)
		}
	}
}

func TestTokensExcludeCommentsAndKeepTextBlocks(t *testing.T) {
	file := analyze(t, "Service.java", fixture)
	for _, token := range file.Tokens {
		if strings.Contains(token.Value, "comment line") || strings.Contains(token.Value, "documentation") {
			t.Fatalf("comment leaked into tokens: %+v", token)
		}
		if token.Type == "TEXT_BLOCK" && token.EndLine != token.Line+2 {
			t.Fatalf("text block should span three lines: %+v", token)
		}
	}
}

func TestSyntaxErrorIsRequiredDiagnostic(t *testing.T) {
	file := Analyze("Broken.java", "/repo/Broken.java", "public class Broken { void m( { } }")
	if len(file.Measurements) != 0 || len(file.Diagnostics) != 1 || file.Diagnostics[0].Code != "java.parse_failed" || !file.Diagnostics[0].IsRequiredError() {
		t.Fatalf("expected a required parse diagnostic, got %+v / %+v", file.Measurements, file.Diagnostics)
	}
}

func TestInterfaceDefaultMethodsRecordsAndEnums(t *testing.T) {
	file := analyze(t, "Shapes.java", `interface Shape {
    double area();
    default String label() { return area() > 1 ? "big" : "small"; }
}
record Point(int x, int y) {
    Point {
        if (x < 0) throw new IllegalArgumentException();
    }
}
enum Color {
    RED { String hex() { return "#f00"; } },
    GREEN;
    String hex() { return "#0f0"; }
}
`)
	want := []string{"Shape.label", "Point.<init>", "RED.hex", "Color.hex"}
	if got := symbols(file); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("symbols = %v, want %v", got, want)
	}
	expect(t, file, domain.MetricCyclomatic, "Shape.label", 2)
	expect(t, file, domain.MetricCyclomatic, "Point.<init>", 2)
}

func contains(items []string, wanted string) bool {
	for _, item := range items {
		if item == wanted {
			return true
		}
	}
	return false
}

func TestImportSpansCoverDeclaredImportsOnly(t *testing.T) {
	spans := Analyze("Spans.java", "/repo/Spans.java", "package sample;\n\nimport java.util.List;\nimport static java.util.Collections.emptyList;\n\npublic class Spans { List<String> names() { return emptyList(); } }\n").ImportSpans
	if !slices.Equal(spans, []adapter.LineSpan{{Line: 3, EndLine: 3}, {Line: 4, EndLine: 4}}) {
		t.Fatalf("declared imports, and only those, must be spans, got %v", spans)
	}
}
