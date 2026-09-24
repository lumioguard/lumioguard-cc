// Package cfamily is the C and C++ adapter: first-branch preprocessing, function
// and control-flow parsing, clone tokens and include-based dependencies.
package cfamily

import (
	"context"
	"fmt"
	"os"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/cfamily/syntax"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/measure"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/sourcetext"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/language"
	"github.com/lumioguard/lumioguard-cc/internal/paths"
)

// Adapter identity. Version changes whenever a measurement algorithm changes.
const (
	ID      = "cfamily"
	Version = "1.0.0"

	ParserName    = "lumioguard-cc/cfamily-parser"
	ParserVersion = "1"

	CognitiveAnalyzerName    = "lumioguard-cc/cognitive-complexity"
	CognitiveAnalyzerVersion = "sonar-spec-v1.5-go-v1"
)

// Exact measurement variants produced by this adapter.
const (
	VariantCyclomatic     = "McCabe-v1-cfamily-with-logical-operators-and-case-labels"
	VariantCognitive      = "sonar-cognitive-complexity-spec-v1.5-go-v1"
	VariantNestingDepth   = "control-flow-depth-v1"
	VariantFunctionLines  = "nonblank-noncomment-source-lines-v1"
	VariantParameterCount = "formal-parameter-slots-v1"
)

var supportedExtensions = language.ExtensionSet(language.C, language.CPlusPlus)

var profile = measure.Profile{
	Variants: adapter.MetricVariants{
		Cyclomatic:     VariantCyclomatic,
		Cognitive:      VariantCognitive,
		NestingDepth:   VariantNestingDepth,
		FunctionLines:  VariantFunctionLines,
		ParameterCount: VariantParameterCount,
	},
	CognitiveAnalyzer:        CognitiveAnalyzerName,
	CognitiveAnalyzerVersion: CognitiveAnalyzerVersion,
}

// Adapter implements adapter.LanguageAdapter for C and C++.
type Adapter struct {
	adapter.Descriptor
}

// New creates the adapter.
func New() *Adapter {
	return &Adapter{Descriptor: adapter.Descriptor{
		AdapterID:        ID,
		AdapterVersion:   Version,
		LanguageNames:    language.Names(language.C, language.CPlusPlus),
		AnalyzerVersions: map[string]string{ParserName: ParserVersion, CognitiveAnalyzerName: CognitiveAnalyzerVersion},
		Extensions:       supportedExtensions,
	}}
}

// Resolver implements adapter.LanguageAdapter.
func (a *Adapter) Resolver() adapter.ImportResolver {
	return includeResolver{}
}

// AnalyzeFile implements adapter.LanguageAdapter.
func (a *Adapter) AnalyzeFile(_ context.Context, root, absolutePath string) (*adapter.SourceFile, error) {
	code, err := os.ReadFile(absolutePath)
	if err != nil {
		return nil, err
	}
	return Analyze(paths.Relative(root, absolutePath), absolutePath, string(code)), nil
}

// Analyze measures source text that is already in memory. A lexical or syntax
// error yields a required diagnostic and no measurements.
func Analyze(relativePath, absolutePath, code string) *adapter.SourceFile {
	file := &adapter.SourceFile{AbsolutePath: absolutePath, RelativePath: relativePath, Code: code}
	lexed, err := syntax.Tokenize(code)
	if err != nil {
		file.Diagnostics = parseFailed(relativePath, err)
		return file
	}
	cpp := language.NameFor(relativePath) == language.CPlusPlus.Name
	functions, failure := parse(lexed.Tokens, cpp)
	if failure != nil {
		file.Diagnostics = parseFailed(relativePath, failure)
		return file
	}
	comments := make([]sourcetext.Range, 0, len(lexed.Comments))
	for _, comment := range lexed.Comments {
		comments = append(comments, sourcetext.Range{Start: comment.Start, End: comment.End})
	}
	file.Tokens = cloneTokens(lexed)
	file.Imports, file.ImportSpans = collectImports(lexed)
	for _, record := range structure.AssignSymbols(relativePath, functions) {
		file.Measurements = append(file.Measurements, measure.Function(code, record, comments, profile)...)
	}
	return file
}

func parseFailed(relativePath string, err error) []domain.Diagnostic {
	return []domain.Diagnostic{domain.RequiredError("cfamily.parse_failed", relativePath,
		fmt.Sprintf("Could not parse source: %v", err))}
}
