package golang

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
)

// collectImports gathers every import path. Go has no dynamic imports. The
// spans cover each import declaration, including a parenthesised block.
func collectImports(p *parsedFile) ([]adapter.Import, []adapter.LineSpan) {
	imports := []adapter.Import{}
	for _, spec := range p.file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		imports = append(imports, adapter.Import{Specifier: path, Line: p.line(spec.Pos()), Kind: adapter.ImportStatic})
	}
	var spans []adapter.LineSpan
	for _, declaration := range p.file.Decls {
		if general, ok := declaration.(*ast.GenDecl); ok && general.Tok == token.IMPORT {
			spans = append(spans, adapter.LineSpan{Line: p.line(general.Pos()), EndLine: p.line(general.End())})
		}
	}
	return imports, spans
}
