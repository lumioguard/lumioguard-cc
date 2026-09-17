// Package adapter defines the contract between the engine and the language
// analyzers, plus the registry that picks an adapter for each file.
package adapter

import (
	"context"
	"maps"
	"slices"

	"github.com/lumioguard/lumioguard-cc/internal/adapter/sourcetext"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

// Token is one lexical token used for exact clone detection. Comments are not
// tokens.
type Token struct {
	Type    string
	Value   string
	Line    int
	EndLine int
	Index   int
}

// ImportKind distinguishes how a dependency was declared.
type ImportKind string

// Supported import declaration kinds.
const (
	// ImportStatic is a declared import, re-export or package import.
	ImportStatic ImportKind = "static"
	// ImportDynamic is a literal runtime import such as import("x") or importlib.import_module("x").
	ImportDynamic ImportKind = "dynamic"
	// ImportRequire is a CommonJS require or a TypeScript import-equals.
	ImportRequire ImportKind = "require"
	// ImportImplicit is a reference that needs no declaration, such as a Java
	// same-package type name.
	ImportImplicit ImportKind = "implicit"
)

// Import is a module specifier found in a source file.
type Import struct {
	Specifier string
	Line      int
	Kind      ImportKind
}

// LineSpan is an inclusive range of 1-based lines.
type LineSpan struct {
	Line    int
	EndLine int
}

// Module describes where a file sits in its language's module system, when
// the language has one (Java packages, Python packages, Go import paths).
type Module struct {
	// Package is the declared or inferred package of the file ("" if none).
	// For Go it is the full import path of the file's package.
	Package string
	// Root is the module the package belongs to, for languages whose modules
	// hold several packages (the Go module path); "" otherwise.
	Root string
}

// SourceFile is the complete analysis result of one file.
type SourceFile struct {
	AbsolutePath string
	RelativePath string
	Code         string
	Module       Module
	Tokens       []Token
	Imports      []Import
	// ImportSpans are the lines of import declarations (imports, re-exports,
	// package imports). Clone detection skips them: files that import the same
	// modules are not copies of each other.
	ImportSpans  []LineSpan
	Measurements []domain.Measurement
	Diagnostics  []domain.Diagnostic
}

// HasRequiredDiagnostic reports whether the file's analysis is incomplete.
func (f *SourceFile) HasRequiredDiagnostic() bool {
	for _, diagnostic := range f.Diagnostics {
		if diagnostic.Required {
			return true
		}
	}
	return false
}

// NonBlankLineCount counts physical lines containing non-whitespace text,
// by the same rule the per-function size metric uses.
func (f *SourceFile) NonBlankLineCount() int {
	return sourcetext.CountSourceLines(f.Code, 0, len(f.Code), nil)
}

// LanguageAdapter is implemented once per language family. It never turns an
// unsupported measurement into zero; failures are required diagnostics.
type LanguageAdapter interface {
	// ID is the stable adapter identifier stored in baselines.
	ID() string
	// Version changes whenever a measurement algorithm changes.
	Version() string
	// Languages lists the human-readable languages handled by the adapter.
	Languages() []string
	// Analyzers lists the underlying parser and analyzer versions.
	Analyzers() map[string]string
	// Supports reports whether the adapter handles the given file name.
	Supports(filename string) bool
	// Resolver maps the adapter's imports onto analyzed files.
	Resolver() ImportResolver
	// AnalyzeFile parses and measures one file. Syntax errors produce
	// diagnostics; only I/O failures return an error.
	AnalyzeFile(ctx context.Context, root, absolutePath string) (*SourceFile, error)
}

// Finder selects the adapter for a file name.
type Finder interface {
	Find(filename string) (LanguageAdapter, bool)
}

// Summary describes an adapter's identity without per-run counts.
func Summary(a LanguageAdapter) domain.AdapterSummary {
	return domain.AdapterSummary{
		ID:        a.ID(),
		Version:   a.Version(),
		Languages: slices.Clone(a.Languages()),
		Analyzers: maps.Clone(a.Analyzers()),
	}
}
