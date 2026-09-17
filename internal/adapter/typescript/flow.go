package typescript

import (
	"slices"

	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/ast"
)

// flowBuilder translates TypeScript AST subtrees into the control-flow model,
// flattening constructs that have no complexity meaning.
type flowBuilder struct {
	p *parsedFile
}

// body builds the root tree of a measured function.
func (b flowBuilder) body(function *ast.Node) *structure.Node {
	return &structure.Node{Kind: structure.KindBlock, Children: b.convert(function.Body())}
}

// nested builds the node of a function-like construct found inside another function.
func (b flowBuilder) nested(function *ast.Node) *structure.Node {
	node := &structure.Node{Kind: structure.KindFunction, Label: "Function", Line: b.p.startLine(function)}
	for _, parameter := range function.Parameters() {
		node.Condition = append(node.Condition, b.convert(parameter)...)
	}
	node.Body = b.convert(function.Body())
	return node
}

func (b flowBuilder) convert(node *ast.Node) []*structure.Node {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case ast.KindIfStatement:
		return structure.One(b.ifStatement(node))
	case ast.KindConditionalExpression:
		expression := node.AsConditionalExpression()
		return structure.One(&structure.Node{
			Kind:      structure.KindTernary,
			Label:     "ConditionalExpression",
			Line:      b.p.startLine(node),
			Condition: b.convert(expression.Condition),
			Body:      b.convert(expression.WhenTrue),
			Else:      b.convert(expression.WhenFalse),
		})
	case ast.KindSwitchStatement:
		return structure.One(b.switchStatement(node))
	case ast.KindForStatement:
		statement := node.AsForStatement()
		return structure.One(&structure.Node{
			Kind:      structure.KindLoop,
			Label:     "ForStatement",
			Line:      b.p.startLine(node),
			Condition: slices.Concat(b.convert(statement.Initializer), b.convert(statement.Condition), b.convert(statement.Incrementor)),
			Body:      b.convert(node.Statement()),
		})
	case ast.KindForInStatement, ast.KindForOfStatement:
		statement := node.AsForInOrOfStatement()
		label := "ForInStatement"
		if node.Kind == ast.KindForOfStatement {
			label = "ForOfStatement"
		}
		return structure.One(&structure.Node{
			Kind:      structure.KindLoop,
			Label:     label,
			Line:      b.p.startLine(node),
			Condition: slices.Concat(b.convert(statement.Initializer), b.convert(statement.Expression)),
			Body:      b.convert(node.Statement()),
		})
	case ast.KindWhileStatement:
		return structure.One(&structure.Node{
			Kind:      structure.KindLoop,
			Label:     "WhileStatement",
			Line:      b.p.startLine(node),
			Condition: b.convert(node.AsWhileStatement().Expression),
			Body:      b.convert(node.Statement()),
		})
	case ast.KindDoStatement:
		return structure.One(&structure.Node{
			Kind:      structure.KindLoop,
			Label:     "DoWhileStatement",
			Line:      b.p.startLine(node),
			Condition: b.convert(node.AsDoStatement().Expression),
			Body:      b.convert(node.Statement()),
		})
	case ast.KindTryStatement:
		return structure.One(b.tryStatement(node))
	case ast.KindBreakStatement, ast.KindContinueStatement:
		label := "BreakStatement"
		if node.Kind == ast.KindContinueStatement {
			label = "ContinueStatement"
		}
		return structure.One(&structure.Node{Kind: structure.KindJump, Label: label, Line: b.p.startLine(node), Labeled: node.Label() != nil})
	case ast.KindBinaryExpression:
		switch node.AsBinaryExpression().OperatorToken.Kind {
		case ast.KindAmpersandAmpersandToken, ast.KindBarBarToken:
			return structure.One(b.logical(node, isSequenceOperator, true))
		case ast.KindQuestionQuestionToken:
			return structure.One(b.logical(node, isCoalesceOperator, false))
		}
		return b.children(node)
	default:
		if ast.IsFunctionLikeDeclaration(node) {
			return structure.One(b.nested(node))
		}
		return b.children(node)
	}
}

func (b flowBuilder) ifStatement(node *ast.Node) *structure.Node {
	statement := node.AsIfStatement()
	out := &structure.Node{
		Kind:      structure.KindIf,
		Label:     "IfStatement",
		Line:      b.p.startLine(node),
		Condition: b.convert(statement.Expression),
		Body:      b.convert(statement.ThenStatement),
	}
	if statement.ElseStatement != nil {
		out.HasElse = true
		out.ElseLine = b.p.startLine(statement.ElseStatement)
		out.Else = b.convert(statement.ElseStatement)
		if statement.ElseStatement.Kind == ast.KindIfStatement && len(out.Else) == 1 {
			out.Else[0].ElseIf = true
		}
	}
	return out
}

func (b flowBuilder) switchStatement(node *ast.Node) *structure.Node {
	statement := node.AsSwitchStatement()
	out := &structure.Node{
		Kind:      structure.KindSwitch,
		Label:     "SwitchStatement",
		Line:      b.p.startLine(node),
		Condition: b.convert(statement.Expression),
	}
	for _, clause := range statement.CaseBlock.AsCaseBlock().Clauses.Nodes {
		data := clause.AsCaseOrDefaultClause()
		caseNode := &structure.Node{
			Kind:      structure.KindCase,
			Label:     "SwitchCase",
			Line:      b.p.startLine(clause),
			Default:   clause.Kind == ast.KindDefaultClause,
			Condition: b.convert(data.Expression),
		}
		for _, statement := range data.Statements.Nodes {
			caseNode.Children = append(caseNode.Children, b.convert(statement)...)
		}
		out.Handlers = append(out.Handlers, caseNode)
	}
	return out
}

func (b flowBuilder) tryStatement(node *ast.Node) *structure.Node {
	statement := node.AsTryStatement()
	out := &structure.Node{
		Kind:    structure.KindTry,
		Label:   "TryStatement",
		Line:    b.p.startLine(node),
		Body:    b.convert(statement.TryBlock),
		Finally: b.convert(statement.FinallyBlock),
	}
	if statement.CatchClause != nil {
		clause := statement.CatchClause.AsCatchClause()
		out.Handlers = []*structure.Node{{
			Kind:      structure.KindCatch,
			Label:     "CatchClause",
			Line:      b.p.startLine(statement.CatchClause),
			Condition: b.convert(clause.VariableDeclaration),
			Body:      b.convert(clause.Block),
		}}
	}
	return out
}

// logical flattens a tree of binary operators accepted by matches, looking
// through parentheses, into one KindLogical node with operators in source order.
func (b flowBuilder) logical(node *ast.Node, matches func(ast.Kind) bool, sequence bool) *structure.Node {
	out := &structure.Node{Kind: structure.KindLogical, Label: "LogicalExpression", Line: b.p.startLine(node)}
	var flatten func(current *ast.Node)
	flatten = func(current *ast.Node) {
		for current.Kind == ast.KindParenthesizedExpression {
			current = current.AsParenthesizedExpression().Expression
		}
		if current.Kind == ast.KindBinaryExpression {
			binary := current.AsBinaryExpression()
			if matches(binary.OperatorToken.Kind) {
				flatten(binary.Left)
				out.Operators = append(out.Operators, structure.Operator{
					Text:     operatorText(binary.OperatorToken.Kind),
					Line:     b.p.startLine(binary.OperatorToken),
					Sequence: sequence,
				})
				flatten(binary.Right)
				return
			}
		}
		out.Children = append(out.Children, b.convert(current)...)
	}
	flatten(node)
	return out
}

func (b flowBuilder) children(node *ast.Node) []*structure.Node {
	var out []*structure.Node
	node.ForEachChild(func(child *ast.Node) bool {
		out = append(out, b.convert(child)...)
		return false
	})
	return out
}

func isSequenceOperator(kind ast.Kind) bool {
	return kind == ast.KindAmpersandAmpersandToken || kind == ast.KindBarBarToken
}

func isCoalesceOperator(kind ast.Kind) bool {
	return kind == ast.KindQuestionQuestionToken
}

func operatorText(kind ast.Kind) string {
	switch kind {
	case ast.KindAmpersandAmpersandToken:
		return "&&"
	case ast.KindBarBarToken:
		return "||"
	default:
		return "??"
	}
}
