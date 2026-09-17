package typescript

import (
	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/ast"
)

// collectImports gathers static imports, re-exports, literal dynamic imports and
// literal require calls. Computed specifiers cannot be resolved and are skipped.
// The spans cover the import, re-export and import-equals declarations.
func collectImports(p *parsedFile) ([]adapter.Import, []adapter.LineSpan) {
	collector := &importCollector{p: p, imports: []adapter.Import{}}
	collector.walk(p.source.AsNode())
	return collector.imports, collector.spans
}

type importCollector struct {
	p       *parsedFile
	imports []adapter.Import
	spans   []adapter.LineSpan
}

func (c *importCollector) walk(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindImportDeclaration:
		c.declaration(node)
		c.record(node, node.AsImportDeclaration().ModuleSpecifier, adapter.ImportStatic)
	case ast.KindExportDeclaration:
		c.reExport(node)
	case ast.KindImportEqualsDeclaration:
		c.importEquals(node)
	case ast.KindCallExpression:
		c.call(node)
	}
	node.ForEachChild(c.walk)
	return false
}

func (c *importCollector) record(node *ast.Node, specifier *ast.Node, kind adapter.ImportKind) {
	if specifier == nil || specifier.Kind != ast.KindStringLiteral {
		return
	}
	c.imports = append(c.imports, adapter.Import{Specifier: specifier.Text(), Line: c.p.startLine(node), Kind: kind})
}

func (c *importCollector) declaration(node *ast.Node) {
	c.spans = append(c.spans, adapter.LineSpan{Line: c.p.startLine(node), EndLine: c.p.endLine(node)})
}

// reExport handles `export ... from "x"`; an export without a module is not an import.
func (c *importCollector) reExport(node *ast.Node) {
	specifier := node.AsExportDeclaration().ModuleSpecifier
	if specifier == nil {
		return
	}
	c.declaration(node)
	c.record(node, specifier, adapter.ImportStatic)
}

func (c *importCollector) importEquals(node *ast.Node) {
	c.declaration(node)
	reference := node.AsImportEqualsDeclaration().ModuleReference
	if reference != nil && reference.Kind == ast.KindExternalModuleReference {
		c.record(node, reference.AsExternalModuleReference().Expression, adapter.ImportRequire)
	}
}

// call handles `import("x")` and `require("x")`, which sit inside code rather
// than declaring a dependency, so they get no span.
func (c *importCollector) call(node *ast.Node) {
	call := node.AsCallExpression()
	arguments := call.Arguments.Nodes
	if len(arguments) == 0 {
		return
	}
	switch {
	case call.Expression.Kind == ast.KindImportKeyword:
		c.record(node, arguments[0], adapter.ImportDynamic)
	case call.Expression.Kind == ast.KindIdentifier && call.Expression.Text() == "require":
		c.record(node, arguments[0], adapter.ImportRequire)
	}
}
