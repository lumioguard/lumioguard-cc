// Package python is the Python 3 adapter: the syntax parser, translation into the
// shared control-flow model, clone tokens and package-based imports.
package python

import (
	"context"
	"fmt"
	"os"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/python/syntax"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/sourcetext"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/language"
	"github.com/lumioguard/lumioguard-cc/internal/paths"
)

// Adapter identity. Version changes whenever a measurement algorithm changes.
const (
	ID      = "python"
	Version = "1.0.0"

	ParserName    = "lumioguard-cc/python-parser"
	ParserVersion = "python3-syntax-go-v1"

	CognitiveAnalyzerName    = "lumioguard-cc/cognitive-complexity"
	CognitiveAnalyzerVersion = "sonar-spec-v1.5-go-v1"
)

// Exact measurement variants produced by this adapter.
const (
	VariantCyclomatic     = "McCabe-v1-python-with-logical-operators-and-comprehensions"
	VariantCognitive      = "sonar-cognitive-complexity-spec-v1.5-go-v1"
	VariantNestingDepth   = "control-flow-depth-v1"
	VariantFunctionLines  = "nonblank-noncomment-source-lines-v1"
	VariantParameterCount = "formal-parameter-slots-v1"
)

var supportedExtensions = language.ExtensionSet(language.Python)

// Adapter implements adapter.LanguageAdapter for Python 3.
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
func (a *Adapter) Languages() []string { return language.Names(language.Python) }

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
	module, tokens, comments, err := syntax.Parse(code)
	if err != nil {
		file.Diagnostics = []domain.Diagnostic{domain.RequiredError("python.parse_failed", relativePath,
			fmt.Sprintf("Could not parse source: %v", err))}
		return file
	}
	commentRanges := make([]sourcetext.Range, len(comments))
	for i, comment := range comments {
		commentRanges[i] = sourcetext.Range{Start: comment.Start, End: comment.End}
	}
	file.Tokens = convertTokens(tokens)
	file.Imports, file.ImportSpans = collectImports(module)
	for _, record := range collectFunctions(module, relativePath) {
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
			SourceLines:    sourcetext.CountSourceLines(code, function.Start, function.End, commentRanges),
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

// convertTokens maps lexer tokens to clone tokens. Newline, indent and dedent
// are kept because they delimit blocks the way braces do.
func convertTokens(tokens []syntax.Token) []adapter.Token {
	out := make([]adapter.Token, 0, len(tokens))
	for _, token := range tokens {
		if token.Type == syntax.EOF {
			continue
		}
		out = append(out, adapter.Token{
			Type:    token.Type.String(),
			Value:   token.Text,
			Line:    token.Line,
			EndLine: token.EndLine,
			Index:   len(out),
		})
	}
	return out
}
