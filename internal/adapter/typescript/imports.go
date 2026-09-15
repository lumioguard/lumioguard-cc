package typescript

import (
	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/thirdparty/tsgo/ast"
)

// collectImports gathers static imports, re-exports, literal dynamic imports and
// literal require calls. Computed specifiers cannot be resolved and are skipped.
func collectImports(p *parsedFile) []adapter.Import {
	imports := []adapter.Import{}
	record := func(node *ast.Node, specifier *ast.Node, kind adapter.ImportKind) {
		if specifier == nil || specifier.Kind != ast.KindStringLiteral {
			return
		}
		imports = append(imports, adapter.Import{Specifier: specifier.Text(), Line: p.startLine(node), Kind: kind})
	}
	var walk func(node *ast.Node) bool
	walk = func(node *ast.Node) bool {
		switch node.Kind {
		case ast.KindImportDeclaration:
			record(node, node.AsImportDeclaration().ModuleSpecifier, adapter.ImportStatic)
		case ast.KindExportDeclaration:
			record(node, node.AsExportDeclaration().ModuleSpecifier, adapter.ImportStatic)
		case ast.KindImportEqualsDeclaration:
			reference := node.AsImportEqualsDeclaration().ModuleReference
			if reference != nil && reference.Kind == ast.KindExternalModuleReference {
				record(node, reference.AsExternalModuleReference().Expression, adapter.ImportRequire)
			}
		case ast.KindCallExpression:
			call := node.AsCallExpression()
			arguments := call.Arguments.Nodes
			if len(arguments) > 0 {
				switch {
				case call.Expression.Kind == ast.KindImportKeyword:
					record(node, arguments[0], adapter.ImportDynamic)
				case call.Expression.Kind == ast.KindIdentifier && call.Expression.Text() == "require":
					record(node, arguments[0], adapter.ImportRequire)
				}
			}
		}
		node.ForEachChild(walk)
		return false
	}
	walk(p.source.AsNode())
	return imports
}
