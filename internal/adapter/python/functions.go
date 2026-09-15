package python

import (
	"github.com/lumiostack/lumioguard-cc/internal/adapter/python/syntax"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/structure"
)

// collectFunctions finds every def, async def and lambda with its enclosing
// class and function, and builds its control-flow tree.
func collectFunctions(module *syntax.Node, relativePath string) []structure.Record {
	builder := flowBuilder{}
	var functions []*structure.Function
	var ancestors []*syntax.Node
	var walk func(node *syntax.Node)
	walk = func(node *syntax.Node) {
		if node.IsFunctionLike() {
			functions = append(functions, &structure.Function{
				Name:          rawFunctionName(node),
				ClassName:     enclosingClassName(ancestors),
				EnclosingName: enclosingFunctionName(ancestors),
				Line:          node.Line,
				EndLine:       node.EndLine,
				Start:         node.Start,
				End:           node.End,
				Parameters:    node.Params,
				Body:          builder.body(node),
			})
		}
		ancestors = append(ancestors, node)
		for _, child := range node.Children() {
			walk(child)
		}
		ancestors = ancestors[:len(ancestors)-1]
	}
	walk(module)
	return structure.AssignSymbols(relativePath, functions)
}

func rawFunctionName(node *syntax.Node) string {
	if node.Kind == syntax.Lambda {
		return "<lambda>"
	}
	return node.Name
}

func enclosingClassName(ancestors []*syntax.Node) string {
	for index := len(ancestors) - 1; index >= 0; index-- {
		if ancestors[index].Kind == syntax.ClassDef {
			return ancestors[index].Name
		}
	}
	return ""
}

func enclosingFunctionName(ancestors []*syntax.Node) string {
	for index := len(ancestors) - 1; index >= 0; index-- {
		if ancestors[index].IsFunctionLike() {
			return rawFunctionName(ancestors[index])
		}
	}
	return ""
}
