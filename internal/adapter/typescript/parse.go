package typescript

import (
	"fmt"
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/paths"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/ast"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/core"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/parser"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/scanner"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/tspath"
)

// parsedFile bundles the AST with the helpers that need the source text.
type parsedFile struct {
	source *ast.SourceFile
	text   string
	jsx    bool
}

func scriptKindFor(filename string) (kind core.ScriptKind, jsx bool) {
	switch paths.Extension(filename) {
	case ".ts", ".mts", ".cts":
		return core.ScriptKindTS, false
	case ".tsx":
		return core.ScriptKindTSX, true
	case ".jsx":
		return core.ScriptKindJSX, true
	default:
		return core.ScriptKindJS, false
	}
}

// parse runs the TypeScript parser. Any syntax diagnostic makes the file
// unanalyzable; the first one is reported with its line.
func parse(relativePath, code string) (*parsedFile, *domain.Diagnostic) {
	kind, jsx := scriptKindFor(relativePath)
	fileName := "/" + strings.TrimPrefix(relativePath, "/")
	options := ast.SourceFileParseOptions{FileName: fileName, Path: tspath.Path(fileName)}
	source := parser.ParseSourceFile(options, code, kind)
	if diagnostics := source.Diagnostics(); len(diagnostics) > 0 {
		first := diagnostics[0]
		line := scanner.GetECMALineOfPosition(source, first.Pos()) + 1
		failure := domain.RequiredError("typescript.parse_failed", relativePath,
			fmt.Sprintf("Could not parse source: %s (line %d)", diagnosticMessage(first), line))
		return nil, &failure
	}
	return &parsedFile{source: source, text: source.Text(), jsx: jsx}, nil
}

func diagnosticMessage(diagnostic *ast.Diagnostic) string {
	if text := diagnostic.MessageText(); text != "" {
		return text
	}
	return strings.ReplaceAll(string(diagnostic.MessageKey()), "_", " ")
}

// line returns the 1-based line of a byte position.
func (p *parsedFile) line(pos int) int {
	return scanner.GetECMALineOfPosition(p.source, pos) + 1
}

// tokenStart skips the leading trivia of a node so that positions refer to
// its first token, as in ESTree ranges.
func (p *parsedFile) tokenStart(node *ast.Node) int {
	return scanner.SkipTrivia(p.text, node.Pos())
}

func (p *parsedFile) startLine(node *ast.Node) int {
	return p.line(p.tokenStart(node))
}

func (p *parsedFile) endLine(node *ast.Node) int {
	end := node.End()
	if end > 0 {
		end--
	}
	return p.line(end)
}

func kindName(kind ast.Kind) string {
	return strings.TrimPrefix(kind.String(), "Kind")
}
