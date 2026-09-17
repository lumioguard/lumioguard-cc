package typescript

import (
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/ast"
)

// collectFunctions finds every function-like node with a body, records its
// enclosing class and function, and builds its control-flow tree.
func collectFunctions(p *parsedFile, relativePath string) []structure.Record {
	builder := flowBuilder{p: p}
	var functions []*structure.Function
	walkWithAncestors(p.source.AsNode(), func(node *ast.Node, ancestors []*ast.Node) {
		if !isMeasuredFunction(node) {
			return
		}
		className := enclosingClassName(ancestors)
		enclosingName := ""
		if index := nearestFunctionIndex(ancestors); index >= 0 {
			enclosingName = rawFunctionName(ancestors[index], ancestors[:index])
		}
		functions = append(functions, &structure.Function{
			Name:          rawFunctionName(node, ancestors),
			ClassName:     className,
			EnclosingName: enclosingName,
			Line:          p.startLine(node),
			EndLine:       p.endLine(node),
			Start:         p.tokenStart(node),
			End:           node.End(),
			Parameters:    len(node.Parameters()),
			Body:          builder.body(node),
		})
	})
	return structure.AssignSymbols(relativePath, functions)
}

// isMeasuredFunction excludes overload signatures and abstract members, which
// have no body and therefore no control flow.
func isMeasuredFunction(node *ast.Node) bool {
	return ast.IsFunctionLikeDeclaration(node) && node.Body() != nil
}

// walkWithAncestors visits every node depth-first in source order. The
// ancestors slice is reused between calls and must not be retained.
func walkWithAncestors(root *ast.Node, visit func(node *ast.Node, ancestors []*ast.Node)) {
	var ancestors []*ast.Node
	var walk func(node *ast.Node) bool
	walk = func(node *ast.Node) bool {
		visit(node, ancestors)
		ancestors = append(ancestors, node)
		node.ForEachChild(walk)
		ancestors = ancestors[:len(ancestors)-1]
		return false
	}
	walk(root)
}

func rawFunctionName(node *ast.Node, ancestors []*ast.Node) string {
	switch node.Kind {
	case ast.KindFunctionDeclaration:
		if name := node.Name(); name != nil {
			return name.Text()
		}
	case ast.KindMethodDeclaration, ast.KindGetAccessor, ast.KindSetAccessor:
		return keyName(node.Name(), "<computed-method>")
	case ast.KindConstructor:
		return "constructor"
	}
	if parent, child := declarationParent(node, ancestors); parent != nil {
		switch parent.Kind {
		case ast.KindVariableDeclaration:
			if parent.Initializer() == child {
				return keyName(parent.Name(), "<destructured>")
			}
		case ast.KindPropertyDeclaration, ast.KindPropertyAssignment:
			if parent.Initializer() == child {
				return keyName(parent.Name(), "<computed-property>")
			}
		}
	}
	if node.Kind == ast.KindFunctionExpression {
		if name := node.Name(); name != nil {
			return name.Text()
		}
	}
	return "<anonymous>"
}

// declarationParent returns the nearest ancestor that is not a parenthesised
// expression, together with the direct child on the path to node.
func declarationParent(node *ast.Node, ancestors []*ast.Node) (*ast.Node, *ast.Node) {
	child := node
	for index := len(ancestors) - 1; index >= 0; index-- {
		if ancestors[index].Kind == ast.KindParenthesizedExpression {
			child = ancestors[index]
			continue
		}
		return ancestors[index], child
	}
	return nil, nil
}

func keyName(name *ast.Node, fallback string) string {
	if name == nil {
		return fallback
	}
	switch name.Kind {
	case ast.KindIdentifier, ast.KindPrivateIdentifier, ast.KindStringLiteral, ast.KindNumericLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return name.Text()
	default:
		return fallback
	}
}

func enclosingClassName(ancestors []*ast.Node) string {
	for index := len(ancestors) - 1; index >= 0; index-- {
		ancestor := ancestors[index]
		if !ast.IsClassLike(ancestor) {
			continue
		}
		if name := ancestor.Name(); name != nil && name.Kind == ast.KindIdentifier {
			return name.Text()
		}
		return "<anonymous-class>"
	}
	return ""
}

func nearestFunctionIndex(ancestors []*ast.Node) int {
	for index := len(ancestors) - 1; index >= 0; index-- {
		if ast.IsFunctionLikeDeclaration(ancestors[index]) {
			return index
		}
	}
	return -1
}
