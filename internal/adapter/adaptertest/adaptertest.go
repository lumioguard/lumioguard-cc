// Package adaptertest holds assertions shared by the language adapter tests.
package adaptertest

import (
	"slices"
	"strings"
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

// Symbols lists the file's function symbols in report order.
func Symbols(file *adapter.SourceFile) []string {
	var symbols []string
	for _, measurement := range file.Measurements {
		if !slices.Contains(symbols, measurement.Scope.Symbol) {
			symbols = append(symbols, measurement.Scope.Symbol)
		}
	}
	return symbols
}

// Value returns the measurement of a function, failing the test if there is none.
func Value(t testing.TB, file *adapter.SourceFile, id domain.MetricID, symbol string) float64 {
	t.Helper()
	for _, measurement := range file.Measurements {
		if measurement.MetricID == id && measurement.Scope.Symbol == symbol {
			return *measurement.Value
		}
	}
	t.Fatalf("no %s measurement for symbol %q; symbols: %v", id, symbol, Symbols(file))
	return 0
}

// Expect checks the measurement of a function.
func Expect(t testing.TB, file *adapter.SourceFile, id domain.MetricID, symbol string, want float64) {
	t.Helper()
	if got := Value(t, file, id, symbol); got != want {
		t.Errorf("%s(%s) = %v, want %v", id, symbol, got, want)
	}
}

// ExpectSymbols checks the function symbols of a file, in order.
func ExpectSymbols(t testing.TB, file *adapter.SourceFile, want ...string) {
	t.Helper()
	if got := Symbols(file); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("symbols = %v, want %v", got, want)
	}
}
