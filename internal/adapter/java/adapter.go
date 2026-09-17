// Package java is the Java adapter: an ANTLR-generated Java 17 parser, translation
// into the shared control-flow model, clone tokens and package-based imports.
package java

import (
	"context"
	"fmt"
	"os"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/sourcetext"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/language"
	"github.com/lumioguard/lumioguard-cc/internal/paths"
)

// Adapter identity. Version changes whenever a measurement algorithm changes.
const (
	ID      = "java"
	Version = "1.0.0"

	ParserName    = "antlr/grammars-v4/java"
	ParserVersion = "942add78ce4402677ebd10ccd35ec07e28710b47+antlr4.13.2"

	CognitiveAnalyzerName    = "lumioguard-cc/cognitive-complexity"
	CognitiveAnalyzerVersion = "sonar-spec-v1.5-go-v1"
)

// Exact measurement variants produced by this adapter.
const (
	VariantCyclomatic     = "McCabe-v1-java-with-logical-operators-and-case-labels"
	VariantCognitive      = "sonar-cognitive-complexity-spec-v1.5-go-v1"
	VariantNestingDepth   = "control-flow-depth-v1"
	VariantFunctionLines  = "nonblank-noncomment-source-lines-v1"
	VariantParameterCount = "formal-parameter-slots-v1"
)

var supportedExtensions = language.ExtensionSet(language.Java)

// Adapter implements adapter.LanguageAdapter for Java.
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
func (a *Adapter) Languages() []string { return language.Names(language.Java) }

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
// yields a required diagnostic and no measurements.
func Analyze(relativePath, absolutePath, code string) *adapter.SourceFile {
	file := &adapter.SourceFile{AbsolutePath: absolutePath, RelativePath: relativePath, Code: code}
	parsed, failure := parse(code)
	if failure != nil {
		file.Diagnostics = []domain.Diagnostic{domain.RequiredError("java.parse_failed", relativePath,
			fmt.Sprintf("Could not parse source: %v", failure))}
		return file
	}
	tokens, comments := tokenize(parsed)
	file.Tokens = tokens
	file.Module = adapter.Module{Package: packageName(parsed)}
	file.Imports, file.ImportSpans = collectImports(parsed)
	for _, record := range collectFunctions(parsed, relativePath) {
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
			SourceLines:    sourcetext.CountSourceLines(code, function.Start, function.End, comments),
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
