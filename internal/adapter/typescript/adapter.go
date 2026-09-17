// Package typescript is the JavaScript and TypeScript adapter over a vendored copy
// of Microsoft's typescript-go parser: control flow, clone tokens and imports.
package typescript

import (
	"context"
	"os"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/sourcetext"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/structure"
	"github.com/lumiostack/lumioguard-cc/internal/domain"
	"github.com/lumiostack/lumioguard-cc/internal/language"
	"github.com/lumiostack/lumioguard-cc/internal/paths"
)

// Adapter identity. Version changes whenever a measurement algorithm changes.
const (
	ID      = "javascript-typescript"
	Version = "1.0.0"

	ParserName    = "microsoft/typescript-go"
	ParserVersion = "v0.0.0-20260820064610-89d5d5b2849a"

	CognitiveAnalyzerName    = "lumioguard-cc/cognitive-complexity"
	CognitiveAnalyzerVersion = "sonar-spec-v1.5-go-v1"
)

// Exact measurement variants produced by this adapter.
const (
	VariantCyclomatic     = "McCabe-v1-with-logical-operators"
	VariantCognitive      = "sonar-cognitive-complexity-spec-v1.5-go-v1"
	VariantNestingDepth   = "control-flow-depth-v1"
	VariantFunctionLines  = "nonblank-noncomment-source-lines-v1"
	VariantParameterCount = "formal-parameter-slots-v1"
)

var (
	languages           = []language.Language{language.JavaScript, language.TypeScript}
	supportedExtensions = language.ExtensionSet(languages...)
)

// Adapter implements adapter.LanguageAdapter for JavaScript and TypeScript.
type Adapter struct {
	resolver *Resolver
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
func (a *Adapter) Languages() []string { return language.Names(languages...) }

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
	return Analyze(paths.Relative(root, absolutePath), absolutePath, string(code)), nil
}

// Analyze measures source text that is already in memory. A syntax error
// yields a required diagnostic and no measurements; it is never a clean result.
func Analyze(relativePath, absolutePath, code string) *adapter.SourceFile {
	file := &adapter.SourceFile{AbsolutePath: absolutePath, RelativePath: relativePath, Code: code}
	parsed, failure := parse(relativePath, code)
	if failure != nil {
		file.Diagnostics = []domain.Diagnostic{*failure}
		return file
	}
	lexemes := tokenize(parsed)
	file.Tokens = lexemes.tokens
	file.Imports, file.ImportSpans = collectImports(parsed)
	for _, record := range collectFunctions(parsed, relativePath) {
		file.Measurements = append(file.Measurements, measure(parsed, record, lexemes.comments)...)
	}
	return file
}

func measure(p *parsedFile, record structure.Record, comments []sourcetext.Range) []domain.Measurement {
	function := record.Function
	cyclomatic := structure.Cyclomatic(function)
	cognitive := structure.Cognitive(function)
	return adapter.FunctionMeasurements(adapter.FunctionMetrics{
		Scope:          record.Scope,
		Cyclomatic:     cyclomatic.Value,
		Decisions:      cyclomatic.Decisions,
		Cognitive:      cognitive.Value,
		Increments:     cognitive.Increments,
		NestingDepth:   structure.NestingDepth(function),
		SourceLines:    sourcetext.CountSourceLines(p.text, function.Start, function.End, comments),
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
	})
}
