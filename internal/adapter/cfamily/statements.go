package cfamily

import (
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
)

// block converts the statements of the braced block opening at open.
func (p *parser) block(open int, where scope) []*structure.Node {
	return p.statements(open+1, p.match[open], where)
}

func (p *parser) statements(from, to int, where scope) []*structure.Node {
	var out []*structure.Node
	for i := from; i < to; {
		next, nodes := p.statement(i, to, where)
		out = append(out, nodes...)
		i = next
	}
	return out
}

// statement converts the statement at i and returns the index after it.
func (p *parser) statement(i, to int, where scope) (int, []*structure.Node) {
	if i >= to {
		p.fail(i, "expected a statement")
	}
	switch {
	case p.is(i, "{"):
		return p.match[i] + 1, p.block(i, where)
	case p.is(i, ";"):
		return i + 1, nil
	case p.isCaseLabel(i):
		node, next := p.caseLabel(i, to, where)
		return next, structure.One(node)
	case p.is(i, "else"):
		p.fail(i, "else without a matching if")
	}
	if control := p.controlStatement(i); control != nil {
		return control(i, to, where)
	}
	return p.simpleStatement(i, to, where)
}

type statementParser func(i, to int, where scope) (int, []*structure.Node)

// controlStatement returns the parser of the control statement at i, or nil.
func (p *parser) controlStatement(i int) statementParser {
	switch {
	case p.is(i, "if"):
		return p.ifStatement
	case p.is(i, "for") || p.is(i, "while"):
		return p.loop
	case p.is(i, "do"):
		return p.doWhile
	case p.is(i, "switch"):
		return p.switchStatement
	case p.is(i, "try") && p.is(i+1, "{") || p.is(i, "__try"):
		return p.tryStatement
	}
	return nil
}

// simpleStatement handles jumps, labels, local types, assembly and expressions.
func (p *parser) simpleStatement(i, to int, where scope) (int, []*structure.Node) {
	switch {
	case p.is(i, "break") || p.is(i, "continue"):
		label := "BreakStatement"
		if p.is(i, "continue") {
			label = "ContinueStatement"
		}
		return p.semicolon(i, to) + 1, structure.One(&structure.Node{Kind: structure.KindJump, Label: label, Line: p.tokens[i].Line})
	case p.is(i, "goto"):
		return p.semicolon(i, to) + 1, structure.One(&structure.Node{
			Kind: structure.KindJump, Label: "GotoStatement", Line: p.tokens[i].Line, Labeled: true,
		})
	case p.isName(i) && p.is(i+1, ":"):
		return i + 2, nil
	case (p.is(i, "asm") || p.is(i, "__asm") || p.is(i, "_asm")) && p.is(i+1, "{"):
		return p.match[i+1] + 1, nil
	case p.isLocalType(i, to):
		return p.declaration(i, to, scope{className: where.className, function: where.function})
	}
	return p.expressionStatement(i, to, where)
}

func (p *parser) condition(i int, keyword string, where scope) (int, []*structure.Node) {
	if !p.is(i, "(") {
		p.fail(i, "expected ( after %s", keyword)
	}
	return p.match[i] + 1, p.expression(i+1, p.match[i], where)
}

func (p *parser) ifStatement(i, to int, where scope) (int, []*structure.Node) {
	node := &structure.Node{Kind: structure.KindIf, Label: "IfStatement", Line: p.tokens[i].Line}
	// Words before the condition are constexpr, consteval or a macro, as in
	// "if FMT_CONSTEXPR20 (x)" or "if EQ(s)", whose arguments are the condition.
	j := i + 1
	for p.is(j, "!") || p.isIdentifier(j) && !p.is(j-1, "consteval") {
		j++
	}
	if !p.is(j-1, "consteval") {
		j, node.Condition = p.condition(j, "if", where)
	}
	j, node.Body = p.statement(j, to, where)
	return p.elseBranch(node, j, to, where)
}

// elseBranch reads an optional else branch of node starting at j.
func (p *parser) elseBranch(node *structure.Node, j, to int, where scope) (int, []*structure.Node) {
	if !p.is(j, "else") {
		return j, structure.One(node)
	}
	node.HasElse = true
	node.ElseLine = p.tokens[j].Line
	elseStart := j + 1
	j, node.Else = p.statement(elseStart, to, where)
	if p.is(elseStart, "if") && len(node.Else) == 1 && node.Else[0].Kind == structure.KindIf {
		node.Else[0].ElseIf = true
	}
	return j, structure.One(node)
}

func (p *parser) loop(i, to int, where scope) (int, []*structure.Node) {
	node := &structure.Node{Kind: structure.KindLoop, Label: "WhileStatement", Line: p.tokens[i].Line}
	j := i + 1
	if p.is(i, "for") {
		node.Label = "ForStatement"
		if p.is(j, "co_await") {
			j++
		}
		if p.is(j, "(") && p.semicolon(j+1, p.match[j]) == p.match[j] {
			node.Label = "RangeForStatement"
		}
	}
	j, node.Condition = p.condition(j, p.tokens[i].Text, where)
	j, node.Body = p.statement(j, to, where)
	return j, structure.One(node)
}

func (p *parser) doWhile(i, to int, where scope) (int, []*structure.Node) {
	node := &structure.Node{Kind: structure.KindLoop, Label: "DoWhileStatement", Line: p.tokens[i].Line}
	j, body := p.statement(i+1, to, where)
	node.Body = body
	if !p.is(j, "while") {
		p.fail(j, "expected while after do body")
	}
	j, node.Condition = p.condition(j+1, "while", where)
	if p.is(j, ";") {
		j++
	}
	return j, structure.One(node)
}

func (p *parser) switchStatement(i, to int, where scope) (int, []*structure.Node) {
	node := &structure.Node{Kind: structure.KindSwitch, Label: "SwitchStatement", Line: p.tokens[i].Line}
	j, condition := p.condition(i+1, "switch", where)
	node.Condition = condition
	if !p.is(j, "{") {
		j, node.Body = p.statement(j, to, where)
		return j, structure.One(node)
	}
	closing := p.match[j]
	var current *structure.Node
	for k := j + 1; k < closing; {
		if p.isCaseLabel(k) {
			current, k = p.caseLabel(k, closing, where)
			node.Handlers = append(node.Handlers, current)
			continue
		}
		next, nodes := p.statement(k, closing, where)
		if current == nil {
			node.Body = append(node.Body, nodes...)
		} else {
			current.Children = append(current.Children, nodes...)
		}
		k = next
	}
	return closing + 1, structure.One(node)
}

func (p *parser) isCaseLabel(i int) bool {
	return p.is(i, "case") || p.is(i, "default") && p.is(i+1, ":")
}

// caseLabel reads "case value:" or "default:". The colon is the first one not
// paired with a "?" of a conditional expression.
func (p *parser) caseLabel(i, to int, where scope) (*structure.Node, int) {
	node := &structure.Node{Kind: structure.KindCase, Label: "SwitchCase", Line: p.tokens[i].Line}
	if p.is(i, "default") {
		node.Default = true
		return node, i + 2
	}
	colon := p.colon(i+1, to)
	if colon >= to {
		p.fail(i, "case label without a colon")
	}
	node.Condition = p.expression(i+1, colon, where)
	return node, colon + 1
}

// colon returns the first top-level ":" in [from, to) that closes no "?".
func (p *parser) colon(from, to int) int {
	pending := 0
	for i := from; i < to; i = p.next(i) {
		switch {
		case p.is(i, "?"):
			pending++
		case p.is(i, ":") && pending > 0:
			pending--
		case p.is(i, ":"):
			return i
		}
	}
	return to
}

func (p *parser) tryStatement(i, to int, where scope) (int, []*structure.Node) {
	if !p.is(i+1, "{") {
		p.fail(i, "expected { after %s", p.tokens[i].Text)
	}
	node := &structure.Node{Kind: structure.KindTry, Label: "TryStatement", Line: p.tokens[i].Line, Body: p.block(i+1, where)}
	handlers, last := p.handlers(p.match[i+1]+1, to, where)
	node.Handlers = handlers
	if p.is(last+1, "__finally") && p.is(last+2, "{") {
		node.Finally = p.block(last+2, where)
		last = p.match[last+2]
	}
	return last + 1, structure.One(node)
}

// handlers reads catch clauses (and MSVC __except) from j and returns them
// with the index of the last token they cover.
func (p *parser) handlers(j, to int, where scope) ([]*structure.Node, int) {
	var out []*structure.Node
	last := j - 1
	for (p.is(j, "catch") || p.is(j, "__except")) && p.is(j+1, "(") && j < to {
		open := p.match[j+1] + 1
		if !p.is(open, "{") {
			p.fail(open, "expected { after %s", p.tokens[j].Text)
		}
		out = append(out, &structure.Node{Kind: structure.KindCatch, Label: "CatchClause", Line: p.tokens[j].Line, Body: p.block(open, where)})
		last = p.match[open]
		j = last + 1
	}
	return out, last
}

// isLocalType reports a class, struct, union or enum definition in a body.
func (p *parser) isLocalType(i, to int) bool {
	if p.is(i, "typedef") {
		i++
	}
	if !p.is(i, "enum") && !p.isClassKey(i) {
		return false
	}
	// "struct tag name {...}" initializes a variable; a local class has no
	// export macro, so a second name before the brace means a variable.
	names := 0
	for j := i + 1; j < to; j = p.next(j) {
		switch {
		case p.is(j, "{"):
			return names < 2
		case p.is(j, ";") || p.is(j, "=") || p.is(j, "("):
			return false
		case p.is(j, ":") || p.is(j, "::"):
			names = 0
		case p.isName(j):
			names++
		}
	}
	return false
}

// expressionStatement reads an expression up to ";". An unknown macro may
// end without one, so a statement keyword or a statement block also ends it.
func (p *parser) expressionStatement(i, to int, where scope) (int, []*structure.Node) {
	for j := i; j < to; j = p.expressionStep(i, j, to) {
		switch {
		case p.is(j, ";"):
			return j + 1, p.expression(i, j, where)
		case j > i && p.startsStatement(j):
			return j, p.expression(i, j, where)
		case p.is(j, "{") && p.isStatementBlock(j):
			return p.macroBlock(i, j, to, where)
		}
	}
	return to, p.expression(i, to, where)
}

// expressionStep skips one token or group of the expression statement
// starting at i, taking a lambda or a requires expression whole.
func (p *parser) expressionStep(i, j, to int) int {
	switch {
	case p.is(j, "[") && (j == i || !p.endsOperand(j-1)):
		if _, closing, ok := p.lambdaParts(j, to); ok {
			return closing + 1
		}
	case p.is(j, "requires"):
		return p.skipRequires(j, to)
	}
	return p.next(j)
}

// startsStatement reports a keyword that can only begin a statement.
func (p *parser) startsStatement(j int) bool {
	return p.isWord(j, statementWords) || p.is(j, "try") && p.is(j+1, "{")
}

// macroBlock reads "MACRO(args) { ... }". Only an if can take an else, so a
// following else proves the macro ends in one; otherwise the block is plain.
func (p *parser) macroBlock(i, open, to int, where scope) (int, []*structure.Node) {
	closing := p.match[open]
	if p.is(closing+1, "else") {
		node := &structure.Node{
			Kind: structure.KindIf, Label: "IfStatement", Line: p.tokens[i].Line,
			Condition: p.expression(i, open, where), Body: p.block(open, where),
		}
		return p.elseBranch(node, closing+1, to, where)
	}
	return closing + 1, append(p.expression(i, open, where), p.block(open, where)...)
}

// isStatementBlock reports a brace group holding statements rather than
// initializers: the body of a loop macro or a GNU statement expression.
func (p *parser) isStatementBlock(open int) bool {
	closing := p.match[open]
	if open+1 == closing {
		return false
	}
	if p.startsStatement(open + 1) {
		return true
	}
	return p.semicolon(open+1, closing) < closing
}
