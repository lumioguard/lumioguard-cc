// Package golang is the Go adapter: the standard library parser, translation
// into the shared control-flow model, clone tokens and module-based imports.
package golang

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/language"
	"github.com/lumioguard/lumioguard-cc/internal/paths"
)

// Adapter identity. Version changes whenever a measurement algorithm changes.
const (
	ID      = "go"
	Version = "1.0.0"

	// ParserName is the standard library parser; its behaviour is fixed by the
	// Go release the binary was built with, which release builds pin.
	ParserName    = "go/parser"
	ParserVersion = "stdlib-go1.27-v1"

	CognitiveAnalyzerName    = "lumioguard-cc/cognitive-complexity"
	CognitiveAnalyzerVersion = "sonar-spec-v1.5-go-v1"
)

// Exact measurement variants produced by this adapter.
const (
	VariantCyclomatic     = "McCabe-v1-go-with-logical-operators-and-select"
	VariantCognitive      = "sonar-cognitive-complexity-spec-v1.5-go-v1"
	VariantNestingDepth   = "control-flow-depth-v1"
	VariantFunctionLines  = "nonblank-noncomment-source-lines-v1"
	VariantParameterCount = "formal-parameter-slots-v1"
)

var supportedExtensions = language.ExtensionSet(language.Go)

// Adapter implements adapter.LanguageAdapter for Go.
type Adapter struct {
	resolver *Resolver
	modules  moduleFinder
}

// New creates the adapter.
func New() *Adapter {
	return &Adapter{resolver: NewResolver()}
}

// ID implements adapter.LanguageAdapter.
func (a *Adapter) ID() string { return ID }

// Version implements adapter.LanguageAdapter.
func (a *Adapter) Version() string { return Version }

// Languages implements adapter.LanguageAdapter.
func (a *Adapter) Languages() []string { return language.Names(language.Go) }

// Analyzers implements adapter.LanguageAdapter.
func (a *Adapter) Analyzers() map[string]string {
	return map[string]string{
		ParserName:            ParserVersion,
		CognitiveAnalyzerName: CognitiveAnalyzerVersion,
	}
}

// Supports implements adapter.LanguageAdapter.
func (a *Adapter) Supports(filename string) bool {
	return supportedExtensions[paths.Extension(filename)]
}

// Resolver implements adapter.LanguageAdapter.
func (a *Adapter) Resolver() adapter.ImportResolver {
	return a.resolver
}

// AnalyzeFile implements adapter.LanguageAdapter.
func (a *Adapter) AnalyzeFile(_ context.Context, root, absolutePath string) (*adapter.SourceFile, error) {
	code, err := os.ReadFile(absolutePath)
	if err != nil {
		return nil, err
	}
	module, err := a.modules.locate(root, absolutePath)
	if err != nil {
		return nil, err
	}
	return Analyze(paths.Relative(root, absolutePath), absolutePath, string(code), module), nil
}

// parsedFile pairs the syntax tree with the position table it was parsed with.
type parsedFile struct {
	fset *token.FileSet
	file *ast.File
}

func (p *parsedFile) line(pos token.Pos) int {
	return p.fset.Position(pos).Line
}

func (p *parsedFile) offset(pos token.Pos) int {
	return p.fset.Position(pos).Offset
}

// Analyze measures source text that is already in memory. A syntax error
// yields a required diagnostic and no measurements.
func Analyze(relativePath, absolutePath, code string, module adapter.Module) *adapter.SourceFile {
	file := &adapter.SourceFile{AbsolutePath: absolutePath, RelativePath: relativePath, Code: code, Module: module}
	fset := token.NewFileSet()
	tree, err := parser.ParseFile(fset, relativePath, code, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		file.Diagnostics = []domain.Diagnostic{domain.RequiredError("go.parse_failed", relativePath,
			fmt.Sprintf("Could not parse source: %v", err))}
		return file
	}
	p := &parsedFile{fset: fset, file: tree}
	tokens, comments := tokenize(code)
	file.Tokens = tokens
	file.Imports, file.ImportSpans = collectImports(p)
	for _, record := range collectFunctions(p, relativePath) {
		function := record.Function
		cyclomatic := structure.Cyclomatic(function)
		cognitive := structure.Cognitive(function)
		file.Measurements = append(file.Measurements, adapter.FunctionMeasurements(adapter.FunctionMetrics{
			Scope:          record.Scope,
			Cyclomatic:     cyclomatic.Value,
			Decisions:      cyclomatic.Decisions,
			Cognitive:      cognitive.Value,
			Increments:     cognitive.Increments,
			NestingDepth:   structure.NestingDepth(function),
			SourceLines:    sourceLines(code, function, comments),
			ParameterCount: function.Parameters,
			Variants: adapter.MetricVariants{
				Cyclomatic:     VariantCyclomatic,
				Cognitive:      VariantCognitive,
				NestingDepth:   VariantNestingDepth,
				FunctionLines:  VariantFunctionLines,
				ParameterCount: VariantParameterCount,
			},
			CognitiveAnalyzer:        CognitiveAnalyzerName,
			CognitiveAnalyzerVersion: CognitiveAnalyzerVersion,
		})...)
	}
	return file
}
