package golang

import (
	"go/ast"
	"slices"

	"github.com/lumioguard/lumioguard-cc/internal/adapter/sourcetext"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
)

// collectFunctions finds every function declaration and function literal with
// a body, records its receiver type and enclosing function, and builds its
// control-flow tree.
func collectFunctions(p *parsedFile, relativePath string) []structure.Record {
	builder := flowBuilder{p: p}
	var functions []*structure.Function
	var ancestors []ast.Node
	ast.Inspect(p.file, func(node ast.Node) bool {
		if node == nil {
			ancestors = ancestors[:len(ancestors)-1]
			return false
		}
		switch fn := node.(type) {
		case *ast.FuncDecl:
			if fn.Body != nil {
				functions = append(functions, &structure.Function{
					Name:          fn.Name.Name,
					ClassName:     receiverTypeName(fn.Recv),
					EnclosingName: enclosingFunctionName(ancestors),
					Line:          p.line(fn.Pos()),
					EndLine:       p.line(fn.End()),
					Start:         p.offset(fn.Pos()),
					End:           p.offset(fn.End()),
					Parameters:    countParameters(fn.Type),
					Body:          builder.body(fn.Body),
				})
			}
		case *ast.FuncLit:
			functions = append(functions, &structure.Function{
				Name:          literalName(fn, ancestors),
				ClassName:     enclosingReceiverName(ancestors),
				EnclosingName: enclosingFunctionName(ancestors),
				Line:          p.line(fn.Pos()),
				EndLine:       p.line(fn.End()),
				Start:         p.offset(fn.Pos()),
				End:           p.offset(fn.End()),
				Parameters:    countParameters(fn.Type),
				Body:          builder.body(fn.Body),
			})
		}
		ancestors = append(ancestors, node)
		return true
	})
	return structure.AssignSymbols(relativePath, functions)
}

// sourceLines counts the function's nonblank, noncomment lines. A doc comment
// sits before the func keyword, so it is outside the range already.
func sourceLines(code string, function *structure.Function, comments []sourcetext.Range) int {
	return sourcetext.CountSourceLines(code, function.Start, function.End, comments)
}

// countParameters counts formal parameter slots: each name in a grouped
// declaration such as (a, b int) counts, an unnamed type counts once, and a
// variadic parameter is one slot. The receiver is not a parameter.
func countParameters(signature *ast.FuncType) int {
	if signature.Params == nil {
		return 0
	}
	count := 0
	for _, field := range signature.Params.List {
		count += max(len(field.Names), 1)
	}
	return count
}

// receiverTypeName returns the base type name of a method receiver, looking
// through pointers and type parameters: (s *Store[K, V]) gives Store.
func receiverTypeName(receiver *ast.FieldList) string {
	if receiver == nil || len(receiver.List) == 0 {
		return ""
	}
	return baseTypeName(receiver.List[0].Type)
}

func baseTypeName(expr ast.Expr) string {
	for {
		switch typed := expr.(type) {
		case *ast.StarExpr:
			expr = typed.X
		case *ast.ParenExpr:
			expr = typed.X
		case *ast.IndexExpr:
			expr = typed.X
		case *ast.IndexListExpr:
			expr = typed.X
		case *ast.Ident:
			return typed.Name
		default:
			return ""
		}
	}
}

// literalName names a function literal after the variable it is assigned to,
// as the TypeScript adapter does, and "<anonymous>" otherwise.
func literalName(fn *ast.FuncLit, ancestors []ast.Node) string {
	if len(ancestors) == 0 {
		return "<anonymous>"
	}
	switch parent := ancestors[len(ancestors)-1].(type) {
	case *ast.AssignStmt:
		if index := operandIndex(parent.Rhs, fn); index >= 0 && index < len(parent.Lhs) {
			if name, ok := parent.Lhs[index].(*ast.Ident); ok {
				return name.Name
			}
		}
	case *ast.ValueSpec:
		if index := operandIndex(parent.Values, fn); index >= 0 && index < len(parent.Names) {
			return parent.Names[index].Name
		}
	}
	return "<anonymous>"
}

// operandIndex returns the position of fn in a right-hand side, or -1.
func operandIndex(values []ast.Expr, fn *ast.FuncLit) int {
	return slices.IndexFunc(values, func(value ast.Expr) bool { return value == ast.Expr(fn) })
}

// enclosingFunctionName returns the raw name of the nearest enclosing
// function declaration or literal, or "".
func enclosingFunctionName(ancestors []ast.Node) string {
	for index := len(ancestors) - 1; index >= 0; index-- {
		switch fn := ancestors[index].(type) {
		case *ast.FuncDecl:
			return fn.Name.Name
		case *ast.FuncLit:
			return literalName(fn, ancestors[:index])
		}
	}
	return ""
}

// enclosingReceiverName returns the receiver type of the method a literal is
// defined in, so nested literals keep the method's prefix.
func enclosingReceiverName(ancestors []ast.Node) string {
	for index := len(ancestors) - 1; index >= 0; index-- {
		if fn, ok := ancestors[index].(*ast.FuncDecl); ok {
			return receiverTypeName(fn.Recv)
		}
	}
	return ""
}
