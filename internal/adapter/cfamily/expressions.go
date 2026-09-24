package cfamily

import (
	"github.com/lumioguard/lumioguard-cc/internal/adapter/cfamily/syntax"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
)

// expression converts [from, to), expressions split by ";" or ",". Only conditionals,
// logical operators, lambdas and statement blocks produce nodes.
func (p *parser) expression(from, to int, where scope) []*structure.Node {
	var out []*structure.Node
	start := from
	for i := from; i <= to; {
		if i == to || p.is(i, ";") || p.is(i, ",") {
			out = append(out, p.assignment(start, i, where)...)
			start = i + 1
			i++
			continue
		}
		i = p.next(i)
	}
	return out
}

// value converts an initializer, naming a lambda after the variable it initializes.
func (p *parser) value(from, to int, name string, where scope) []*structure.Node {
	if name != "" && p.is(from, "[") {
		if _, closing, ok := p.lambdaParts(from, to); ok && closing == to-1 {
			return structure.One(p.lambda(from, to, name, where))
		}
	}
	return p.expression(from, to, where)
}

// assignment splits at the first top-level "?" or assignment operator, which
// bind loosest and group from the right.
func (p *parser) assignment(from, to int, where scope) []*structure.Node {
	for i := from; i < to; i = p.next(i) {
		token := p.tokens[i]
		if token.Kind != syntax.Punct {
			continue
		}
		if token.Text == "?" {
			return p.conditional(from, i, to, where)
		}
		if assignmentOperators[token.Text] {
			left := p.logical(from, i, where)
			return append(left, p.value(i+1, to, p.nameBefore(i), where)...)
		}
	}
	return p.logical(from, to, where)
}

func (p *parser) conditional(from, question, to int, where scope) []*structure.Node {
	colon := p.colon(question+1, to)
	if colon >= to {
		return p.logical(from, to, where)
	}
	return structure.One(&structure.Node{
		Kind:      structure.KindTernary,
		Label:     "ConditionalExpression",
		Line:      p.tokens[from].Line,
		Condition: p.logical(from, question, where),
		Body:      p.expression(question+1, colon, where),
		Else:      p.assignment(colon+1, to, where),
	})
}

// logical builds one node for the top-level && and || operators of [from, to),
// looking through parenthesised operands that are themselves logical.
func (p *parser) logical(from, to int, where scope) []*structure.Node {
	if len(p.logicalOperators(from, to)) == 0 {
		return p.operand(from, to, where)
	}
	node := &structure.Node{Kind: structure.KindLogical, Label: "LogicalExpression", Line: p.tokens[from].Line}
	p.flatten(node, from, to, where)
	return structure.One(node)
}

func (p *parser) flatten(node *structure.Node, from, to int, where scope) {
	start := from
	for _, operator := range append(p.logicalOperators(from, to), to) {
		if p.is(start, "(") && p.match[start] == operator-1 && p.isPureLogical(start+1, operator-1) {
			p.flatten(node, start+1, operator-1, where)
		} else {
			node.Children = append(node.Children, p.operand(start, operator, where)...)
		}
		if operator < to {
			node.Operators = append(node.Operators, structure.Operator{
				Text: spelled(p.tokens[operator].Text), Line: p.tokens[operator].Line, Sequence: true,
			})
		}
		start = operator + 1
	}
}

// isPureLogical reports a range that is a logical expression and nothing looser.
func (p *parser) isPureLogical(from, to int) bool {
	for i := from; i < to; i = p.next(i) {
		if p.is(i, "?") || p.is(i, ",") || p.is(i, ";") || assignmentOperators[p.tokens[i].Text] && p.tokens[i].Kind == syntax.Punct {
			return false
		}
	}
	return len(p.logicalOperators(from, to)) > 0
}

func spelled(text string) string {
	switch text {
	case "and":
		return "&&"
	case "or":
		return "||"
	}
	return text
}

// logicalOperators returns the top-level binary && and || of [from, to).
func (p *parser) logicalOperators(from, to int) []int {
	var out []int
	for i := from; i < to; i = p.next(i) {
		if p.isLogicalOperator(i, from, to) {
			out = append(out, i)
		}
	}
	return out
}

// isLogicalOperator tells a binary && or || from a unary && taking a label's
// address and from a reference declarator such as "auto&& x".
func (p *parser) isLogicalOperator(i, from, to int) bool {
	if !p.isLogicalSpelling(i) || i == from || i+1 >= to || !p.endsOperand(i-1) || !p.startsOperand(i+1) {
		return false
	}
	return !p.is(i, "&&") || !p.isReferenceDeclarator(i, to)
}

func (p *parser) isLogicalSpelling(i int) bool {
	token := p.tokens[i]
	if token.Kind == syntax.Punct {
		return token.Text == "&&" || token.Text == "||"
	}
	return p.cpp && (token.Text == "and" || token.Text == "or")
}

// isReferenceDeclarator reports "&&" after a type keyword or before "name =", "name {" or a
// range-for "name :". The "=" may lie past to; a ":" past to belongs to a conditional.
func (p *parser) isReferenceDeclarator(i, to int) bool {
	if p.isWord(i-1, typeKeywords) {
		return true
	}
	return p.isName(i+1) && (p.is(i+2, "=") || p.is(i+2, "{") || i+2 < to && p.is(i+2, ":"))
}

func (p *parser) endsOperand(i int) bool {
	if i < 0 {
		return false
	}
	token := p.tokens[i]
	switch token.Kind {
	case syntax.Identifier:
		return !operandEnders[token.Text]
	case syntax.Number, syntax.String:
		return true
	}
	switch token.Text {
	case ")", "]", "}", ">", "++", "--":
		return true
	}
	return false
}

func (p *parser) startsOperand(i int) bool {
	token := p.tokens[i]
	if token.Kind != syntax.Punct {
		return true
	}
	switch token.Text {
	case "(", "[", "{", "!", "~", "-", "+", "*", "&", "::", "++", "--":
		return true
	}
	return false
}

// operand converts a range with no looser operator: only its groups matter.
func (p *parser) operand(from, to int, where scope) []*structure.Node {
	var out []*structure.Node
	for i := from; i < to; {
		switch {
		case p.is(i, "[") && p.is(i+1, "["):
			i = p.match[i] + 1
		case p.is(i, "[") && (i == from || !p.endsOperand(i-1)):
			if _, closing, ok := p.lambdaParts(i, to); ok {
				out = append(out, p.lambda(i, to, "", where))
				i = closing + 1
				continue
			}
			out = append(out, p.expression(i+1, p.match[i], where)...)
			i = p.match[i] + 1
		case p.is(i, "{") && p.isStatementBlock(i):
			out = append(out, p.block(i, where)...)
			i = p.match[i] + 1
		case p.isOpen(i):
			out = append(out, p.expression(i+1, p.match[i], where)...)
			i = p.match[i] + 1
		case p.is(i, "requires"):
			i = p.skipRequires(i, to)
		default:
			i++
		}
	}
	return out
}

// skipRequires steps over a requires expression "requires (params) { ... }";
// its requirements are declarations, not code that runs.
func (p *parser) skipRequires(i, to int) int {
	j := i + 1
	if p.is(j, "(") {
		j = p.match[j] + 1
	}
	if j < to && p.is(j, "{") {
		return p.match[j] + 1
	}
	return i + 1
}
