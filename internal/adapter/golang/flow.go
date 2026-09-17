package golang

import (
	"go/ast"
	"go/token"
	"slices"

	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
)

// flowBuilder translates Go syntax trees into the language-neutral
// control-flow model. Constructs without complexity semantics are flattened.
type flowBuilder struct {
	p *parsedFile
}

// body builds the root tree of a measured function.
func (b flowBuilder) body(block *ast.BlockStmt) *structure.Node {
	return &structure.Node{Kind: structure.KindBlock, Children: b.statements(block.List)}
}

func (b flowBuilder) statements(list []ast.Stmt) []*structure.Node {
	var out []*structure.Node
	for _, statement := range list {
		out = append(out, b.convert(statement)...)
	}
	return out
}

func (b flowBuilder) convert(node ast.Node) []*structure.Node {
	if isNil(node) {
		return nil
	}
	if statement, ok := node.(ast.Stmt); ok {
		if out, handled := b.statement(statement); handled {
			return out
		}
	}
	switch n := node.(type) {
	case *ast.FuncLit:
		return structure.One(&structure.Node{
			Kind:  structure.KindFunction,
			Label: "FuncLit",
			Line:  b.p.line(n.Pos()),
			Body:  b.statements(n.Body.List),
		})
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			return structure.One(b.logical(n))
		}
	case *ast.ParenExpr:
		return b.convert(n.X)
	}
	return b.children(node)
}

// statement converts the statements that carry control flow or wrap other
// statements; every other statement is transparent.
func (b flowBuilder) statement(node ast.Stmt) ([]*structure.Node, bool) {
	switch n := node.(type) {
	case *ast.IfStmt:
		return structure.One(b.ifStatement(n)), true
	case *ast.ForStmt:
		return structure.One(&structure.Node{
			Kind:      structure.KindLoop,
			Label:     "ForStmt",
			Line:      b.p.line(n.Pos()),
			Condition: slices.Concat(b.convert(n.Init), b.convert(n.Cond), b.convert(n.Post)),
			Body:      b.statements(n.Body.List),
		}), true
	case *ast.RangeStmt:
		return structure.One(&structure.Node{
			Kind:      structure.KindLoop,
			Label:     "RangeStmt",
			Line:      b.p.line(n.Pos()),
			Condition: slices.Concat(b.convert(n.Key), b.convert(n.Value), b.convert(n.X)),
			Body:      b.statements(n.Body.List),
		}), true
	case *ast.SwitchStmt:
		return structure.One(b.switchStatement("SwitchStmt", n.Pos(), slices.Concat(b.convert(n.Init), b.convert(n.Tag)), n.Body)), true
	case *ast.TypeSwitchStmt:
		return structure.One(b.switchStatement("TypeSwitchStmt", n.Pos(), slices.Concat(b.convert(n.Init), b.convert(n.Assign)), n.Body)), true
	case *ast.SelectStmt:
		return structure.One(b.switchStatement("SelectStmt", n.Pos(), nil, n.Body)), true
	case *ast.BranchStmt:
		return b.branch(n), true
	case *ast.BlockStmt:
		return b.statements(n.List), true
	case *ast.LabeledStmt:
		return b.convert(n.Stmt), true
	default:
		return nil, false
	}
}

func (b flowBuilder) ifStatement(n *ast.IfStmt) *structure.Node {
	out := &structure.Node{
		Kind:      structure.KindIf,
		Label:     "IfStmt",
		Line:      b.p.line(n.Pos()),
		Condition: slices.Concat(b.convert(n.Init), b.convert(n.Cond)),
		Body:      b.statements(n.Body.List),
	}
	if n.Else != nil {
		out.HasElse = true
		out.ElseLine = b.p.line(n.Else.Pos())
		out.Else = b.convert(n.Else)
		if _, chained := n.Else.(*ast.IfStmt); chained && len(out.Else) == 1 {
			out.Else[0].ElseIf = true
		}
	}
	return out
}

// switchStatement covers switch, type switch and select: each clause is a
// case, and a clause without a list is the default.
func (b flowBuilder) switchStatement(label string, pos token.Pos, condition []*structure.Node, body *ast.BlockStmt) *structure.Node {
	out := &structure.Node{Kind: structure.KindSwitch, Label: label, Line: b.p.line(pos), Condition: condition}
	for _, statement := range body.List {
		switch clause := statement.(type) {
		case *ast.CaseClause:
			caseNode := &structure.Node{Kind: structure.KindCase, Label: "CaseClause", Line: b.p.line(clause.Pos()), Default: clause.List == nil}
			for _, expression := range clause.List {
				caseNode.Condition = append(caseNode.Condition, b.convert(expression)...)
			}
			caseNode.Children = b.statements(clause.Body)
			out.Handlers = append(out.Handlers, caseNode)
		case *ast.CommClause:
			out.Handlers = append(out.Handlers, &structure.Node{
				Kind:      structure.KindCase,
				Label:     "CommClause",
				Line:      b.p.line(clause.Pos()),
				Default:   clause.Comm == nil,
				Condition: b.convert(clause.Comm),
				Children:  b.statements(clause.Body),
			})
		}
	}
	return out
}

// branch maps break and continue to jumps, labelled when they name a label;
// goto always names one. Fallthrough adds no path and is dropped.
func (b flowBuilder) branch(n *ast.BranchStmt) []*structure.Node {
	switch n.Tok {
	case token.BREAK, token.CONTINUE:
		return structure.One(&structure.Node{Kind: structure.KindJump, Label: n.Tok.String(), Line: b.p.line(n.Pos()), Labeled: n.Label != nil})
	case token.GOTO:
		return structure.One(&structure.Node{Kind: structure.KindJump, Label: "goto", Line: b.p.line(n.Pos()), Labeled: true})
	default:
		return nil
	}
}

// logical flattens a tree of && and || operators, looking through
// parentheses, into one node with the operators in source order.
func (b flowBuilder) logical(node *ast.BinaryExpr) *structure.Node {
	out := &structure.Node{Kind: structure.KindLogical, Label: "BinaryExpr", Line: b.p.line(node.Pos())}
	var flatten func(current ast.Expr)
	flatten = func(current ast.Expr) {
		for {
			paren, ok := current.(*ast.ParenExpr)
			if !ok {
				break
			}
			current = paren.X
		}
		if binary, ok := current.(*ast.BinaryExpr); ok && (binary.Op == token.LAND || binary.Op == token.LOR) {
			flatten(binary.X)
			out.Operators = append(out.Operators, structure.Operator{Text: binary.Op.String(), Line: b.p.line(binary.OpPos), Sequence: true})
			flatten(binary.Y)
			return
		}
		out.Children = append(out.Children, b.convert(current)...)
	}
	flatten(node)
	return out
}

// children converts the direct children of a transparent node in source order.
func (b flowBuilder) children(node ast.Node) []*structure.Node {
	var out []*structure.Node
	ast.Inspect(node, func(child ast.Node) bool {
		if child == nil {
			return false
		}
		if child == node {
			return true
		}
		out = append(out, b.convert(child)...)
		return false
	})
	return out
}

// isNil reports whether an optional syntax field is absent. Absent statements
// and expressions arrive as nil interfaces; absent blocks as nil pointers.
func isNil(node ast.Node) bool {
	if node == nil {
		return true
	}
	block, ok := node.(*ast.BlockStmt)
	return ok && block == nil
}
