package java

import (
	"slices"

	"github.com/antlr4-go/antlr/v4"

	"github.com/lumioguard/lumioguard-cc/internal/adapter/java/syntax"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
)

// flowBuilder translates Java parse-tree subtrees into the language-neutral
// control-flow model. Constructs without complexity semantics are flattened.
type flowBuilder struct {
	p *parsedFile
}

// body builds the root tree of a measured function.
func (b flowBuilder) body(body antlr.Tree) *structure.Node {
	return &structure.Node{Kind: structure.KindBlock, Children: b.convert(body)}
}

func (b flowBuilder) convert(node antlr.Tree) []*structure.Node {
	if node == nil {
		return nil
	}
	switch ctx := node.(type) {
	case antlr.TerminalNode:
		return nil
	case *syntax.StatementContext:
		return b.statement(ctx)
	case *syntax.BinaryOperatorExpressionContext:
		if operator := ctx.GetBop(); operator != nil && isSequenceOperator(operator.GetText()) {
			return structure.One(b.logical(ctx))
		}
		return b.children(ctx)
	case *syntax.TernaryExpressionContext:
		return structure.One(&structure.Node{
			Kind:      structure.KindTernary,
			Label:     "ConditionalExpression",
			Line:      lineOf(ctx),
			Condition: b.convert(ctx.Expression(0)),
			Body:      b.convert(ctx.Expression(1)),
			Else:      b.convert(ctx.Expression(2)),
		})
	case *syntax.SwitchExpressionContext:
		return structure.One(b.switchExpression(ctx))
	case *syntax.LambdaExpressionContext:
		return structure.One(&structure.Node{
			Kind:  structure.KindFunction,
			Label: "LambdaExpression",
			Line:  lineOf(ctx),
			Body:  b.convert(ctx.LambdaBody()),
		})
	case *syntax.MethodDeclarationContext, *syntax.InterfaceCommonBodyDeclarationContext,
		*syntax.ConstructorDeclarationContext, *syntax.CompactConstructorDeclarationContext:
		_, body, _, _ := functionParts(ctx.(antlr.ParserRuleContext))
		return structure.One(&structure.Node{
			Kind:  structure.KindFunction,
			Label: "MethodDeclaration",
			Line:  lineOf(ctx.(antlr.ParserRuleContext)),
			Body:  b.convert(body),
		})
	default:
		return b.children(node)
	}
}

func (b flowBuilder) statement(ctx *syntax.StatementContext) []*structure.Node {
	switch {
	case ctx.IF() != nil:
		statements := ctx.AllStatement()
		out := &structure.Node{
			Kind:      structure.KindIf,
			Label:     "IfStatement",
			Line:      lineOf(ctx),
			Condition: b.convert(ctx.Expression(0)),
		}
		if len(statements) > 0 {
			out.Body = b.convert(statements[0])
		}
		if len(statements) > 1 {
			elseStatement := statements[1]
			out.HasElse = true
			out.ElseLine = lineOf(elseStatement)
			out.Else = b.convert(elseStatement)
			if isIfStatement(elseStatement) && len(out.Else) == 1 {
				out.Else[0].ElseIf = true
			}
		}
		return structure.One(out)
	case ctx.FOR() != nil:
		label := "ForStatement"
		if control := ctx.ForControl(); control != nil && control.EnhancedForControl() != nil {
			label = "EnhancedForStatement"
		}
		return structure.One(&structure.Node{
			Kind:      structure.KindLoop,
			Label:     label,
			Line:      lineOf(ctx),
			Condition: b.convert(ctx.ForControl()),
			Body:      b.convert(ctx.Statement(0)),
		})
	case ctx.DO() != nil:
		return structure.One(&structure.Node{
			Kind:      structure.KindLoop,
			Label:     "DoWhileStatement",
			Line:      lineOf(ctx),
			Condition: b.convert(ctx.Expression(0)),
			Body:      b.convert(ctx.Statement(0)),
		})
	case ctx.WHILE() != nil:
		return structure.One(&structure.Node{
			Kind:      structure.KindLoop,
			Label:     "WhileStatement",
			Line:      lineOf(ctx),
			Condition: b.convert(ctx.Expression(0)),
			Body:      b.convert(ctx.Statement(0)),
		})
	case ctx.TRY() != nil:
		out := &structure.Node{
			Kind:    structure.KindTry,
			Label:   "TryStatement",
			Line:    lineOf(ctx),
			Body:    slices.Concat(b.convert(ctx.ResourceSpecification()), b.convert(ctx.Block())),
			Finally: b.convert(ctx.FinallyBlock()),
		}
		for _, clause := range ctx.AllCatchClause() {
			out.Handlers = append(out.Handlers, &structure.Node{
				Kind:  structure.KindCatch,
				Label: "CatchClause",
				Line:  lineOf(clause),
				Body:  b.convert(clause.Block()),
			})
		}
		return structure.One(out)
	case ctx.SWITCH() != nil:
		return structure.One(b.switchStatement(ctx))
	case ctx.BREAK() != nil:
		return structure.One(&structure.Node{Kind: structure.KindJump, Label: "BreakStatement", Line: lineOf(ctx), Labeled: ctx.Identifier() != nil})
	case ctx.CONTINUE() != nil:
		return structure.One(&structure.Node{Kind: structure.KindJump, Label: "ContinueStatement", Line: lineOf(ctx), Labeled: ctx.Identifier() != nil})
	default:
		return b.children(ctx)
	}
}

func isIfStatement(statement syntax.IStatementContext) bool {
	ctx, ok := statement.(*syntax.StatementContext)
	return ok && ctx.IF() != nil
}

func (b flowBuilder) switchStatement(ctx *syntax.StatementContext) *structure.Node {
	out := &structure.Node{
		Kind:      structure.KindSwitch,
		Label:     "SwitchStatement",
		Line:      lineOf(ctx),
		Condition: b.convert(ctx.Expression(0)),
	}
	for _, group := range ctx.AllSwitchBlockStatementGroup() {
		labels := group.AllSwitchLabel()
		for index, label := range labels {
			caseNode := b.switchLabel(label)
			if index == len(labels)-1 {
				for _, statement := range group.AllBlockStatement() {
					caseNode.Children = append(caseNode.Children, b.convert(statement)...)
				}
			}
			out.Handlers = append(out.Handlers, caseNode)
		}
	}
	for _, label := range ctx.AllSwitchLabel() {
		out.Handlers = append(out.Handlers, b.switchLabel(label))
	}
	return out
}

func (b flowBuilder) switchLabel(label syntax.ISwitchLabelContext) *structure.Node {
	return &structure.Node{
		Kind:      structure.KindCase,
		Label:     "SwitchCase",
		Line:      lineOf(label),
		Default:   label.DEFAULT() != nil,
		Condition: b.convert(label.Expression()),
	}
}

func (b flowBuilder) switchExpression(ctx *syntax.SwitchExpressionContext) *structure.Node {
	out := &structure.Node{
		Kind:      structure.KindSwitch,
		Label:     "SwitchExpression",
		Line:      lineOf(ctx),
		Condition: b.convert(ctx.Expression()),
	}
	for _, rule := range ctx.AllSwitchLabeledRule() {
		caseNode := &structure.Node{
			Kind:      structure.KindCase,
			Label:     "SwitchCase",
			Line:      lineOf(rule),
			Default:   rule.DEFAULT() != nil,
			Condition: slices.Concat(b.convert(rule.ExpressionList()), b.convert(rule.Guard())),
		}
		caseNode.Children = b.convert(rule.SwitchRuleOutcome())
		out.Handlers = append(out.Handlers, caseNode)
	}
	return out
}

// logical flattens a tree of && and || operators, looking through
// parentheses, into one KindLogical node with operators in source order.
func (b flowBuilder) logical(ctx *syntax.BinaryOperatorExpressionContext) *structure.Node {
	out := &structure.Node{Kind: structure.KindLogical, Label: "LogicalExpression", Line: lineOf(ctx)}
	var flatten func(expression syntax.IExpressionContext)
	flatten = func(expression syntax.IExpressionContext) {
		expression = unwrapParentheses(expression)
		if binary, ok := expression.(*syntax.BinaryOperatorExpressionContext); ok {
			if operator := binary.GetBop(); operator != nil && isSequenceOperator(operator.GetText()) {
				flatten(binary.Expression(0))
				out.Operators = append(out.Operators, structure.Operator{Text: operator.GetText(), Line: operator.GetLine(), Sequence: true})
				flatten(binary.Expression(1))
				return
			}
		}
		out.Children = append(out.Children, b.convert(expression)...)
	}
	flatten(ctx)
	return out
}

// unwrapParentheses looks through "( expression )" primaries.
func unwrapParentheses(expression syntax.IExpressionContext) syntax.IExpressionContext {
	for {
		primaryExpression, ok := expression.(*syntax.PrimaryExpressionContext)
		if !ok {
			return expression
		}
		primary, ok := primaryExpression.Primary().(*syntax.PrimaryContext)
		if !ok || primary.Expression() == nil {
			return expression
		}
		expression = primary.Expression()
	}
}

func (b flowBuilder) children(node antlr.Tree) []*structure.Node {
	var out []*structure.Node
	for _, child := range node.GetChildren() {
		out = append(out, b.convert(child)...)
	}
	return out
}

func isSequenceOperator(text string) bool {
	return text == "&&" || text == "||"
}
