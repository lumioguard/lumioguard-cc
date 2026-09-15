package syntax

// parseStarExpressions parses a comma-separated expression list, which is a
// tuple when a comma is present.
func (p *Parser) parseStarExpressions() *Node {
	start := p.mark()
	first := p.parseStarExpression()
	if !p.atOp(",") {
		return first
	}
	tuple := &Node{Kind: Tuple, Values: []*Node{first}}
	for p.acceptOp(",") {
		if !p.canStartExpression() {
			break
		}
		tuple.Values = append(tuple.Values, p.parseStarExpression())
	}
	return p.finish(tuple, start)
}

func (p *Parser) parseStarExpression() *Node {
	if p.atOp("*") {
		start := p.mark()
		p.next()
		return p.finish(&Node{Kind: Starred, Values: one(p.parseBitOr())}, start)
	}
	return p.parseNamedExpr()
}

func (p *Parser) parseStarOrExpression() *Node {
	if p.atOp("*") {
		start := p.mark()
		p.next()
		return p.finish(&Node{Kind: Starred, Values: one(p.parseBitOr())}, start)
	}
	return p.parseExpression()
}

func (p *Parser) parseNamedExpr() *Node {
	if p.at(Name) && p.peekAt(1).Type == Op && p.peekAt(1).Text == ":=" {
		start := p.mark()
		target := p.finish(&Node{Kind: NameNode, Name: p.next().Text}, start)
		p.next()
		return p.finish(&Node{Kind: NamedExpr, Target: target, Values: one(p.parseExpression())}, start)
	}
	if p.atKeyword("yield") {
		return p.parseYield()
	}
	return p.parseExpression()
}

func (p *Parser) parseExpression() *Node {
	if p.atKeyword("lambda") {
		return p.parseLambda()
	}
	start := p.mark()
	body := p.parseDisjunction()
	if p.atKeyword("if") {
		p.next()
		node := &Node{Kind: IfExp, Body: one(body), Test: p.parseDisjunction()}
		p.expectKeyword("else")
		node.OrElse = one(p.parseExpression())
		return p.finish(node, start)
	}
	return body
}

func (p *Parser) parseLambda() *Node {
	start := p.mark()
	p.expectKeyword("lambda")
	node := &Node{Kind: Lambda}
	node.Params, node.Values = p.parseParameters(":", false)
	p.expectOp(":")
	node.Body = one(p.parseExpression())
	return p.finish(node, start)
}

func (p *Parser) parseDisjunction() *Node {
	start := p.mark()
	left := p.parseConjunction()
	if !p.atKeyword("or") {
		return left
	}
	node := &Node{Kind: BoolOp, Op: "or", Values: []*Node{left}}
	for p.atKeyword("or") {
		node.OpLines = append(node.OpLines, p.next().Line)
		node.Values = append(node.Values, p.parseConjunction())
	}
	return p.finish(node, start)
}

func (p *Parser) parseConjunction() *Node {
	start := p.mark()
	left := p.parseInversion()
	if !p.atKeyword("and") {
		return left
	}
	node := &Node{Kind: BoolOp, Op: "and", Values: []*Node{left}}
	for p.atKeyword("and") {
		node.OpLines = append(node.OpLines, p.next().Line)
		node.Values = append(node.Values, p.parseInversion())
	}
	return p.finish(node, start)
}

func (p *Parser) parseInversion() *Node {
	if p.atKeyword("not") {
		start := p.mark()
		p.next()
		return p.finish(&Node{Kind: UnaryOp, Op: "not", Values: one(p.parseInversion())}, start)
	}
	return p.parseComparison()
}

func (p *Parser) parseComparison() *Node {
	start := p.mark()
	left := p.parseBitOr()
	var node *Node
	for {
		operator, ok := p.comparisonOperator()
		if !ok {
			break
		}
		if node == nil {
			node = &Node{Kind: Compare, Left: left}
		}
		node.Op = operator
		node.Values = append(node.Values, p.parseBitOr())
	}
	if node == nil {
		return left
	}
	return p.finish(node, start)
}

func (p *Parser) comparisonOperator() (string, bool) {
	token := p.peek()
	switch {
	case token.Type == Op && comparisonOperators[token.Text]:
		p.next()
		return token.Text, true
	case token.Type == Name && token.Text == "in":
		p.next()
		return "in", true
	case token.Type == Name && token.Text == "not" && p.peekAt(1).Type == Name && p.peekAt(1).Text == "in":
		p.next()
		p.next()
		return "not in", true
	case token.Type == Name && token.Text == "is":
		p.next()
		if p.acceptKeyword("not") {
			return "is not", true
		}
		return "is", true
	}
	return "", false
}

func (p *Parser) parseBinaryLevel(operators []string, operand func() *Node) *Node {
	start := p.mark()
	left := operand()
	for {
		matched := ""
		for _, operator := range operators {
			if p.atOp(operator) {
				matched = operator
				break
			}
		}
		if matched == "" {
			return left
		}
		p.next()
		left = p.finish(&Node{Kind: BinOp, Op: matched, Left: left, Values: one(operand())}, start)
	}
}

func (p *Parser) parseBitOr() *Node {
	return p.parseBinaryLevel([]string{"|"}, p.parseBitXor)
}

func (p *Parser) parseBitXor() *Node {
	return p.parseBinaryLevel([]string{"^"}, p.parseBitAnd)
}

func (p *Parser) parseBitAnd() *Node {
	return p.parseBinaryLevel([]string{"&"}, p.parseShift)
}

func (p *Parser) parseShift() *Node {
	return p.parseBinaryLevel([]string{"<<", ">>"}, p.parseSum)
}

func (p *Parser) parseSum() *Node {
	return p.parseBinaryLevel([]string{"+", "-"}, p.parseTerm)
}

func (p *Parser) parseTerm() *Node {
	return p.parseBinaryLevel([]string{"*", "/", "//", "%", "@"}, p.parseFactor)
}

func (p *Parser) parseFactor() *Node {
	if p.atOp("+") || p.atOp("-") || p.atOp("~") {
		start := p.mark()
		operator := p.next().Text
		return p.finish(&Node{Kind: UnaryOp, Op: operator, Values: one(p.parseFactor())}, start)
	}
	return p.parsePower()
}

func (p *Parser) parsePower() *Node {
	start := p.mark()
	base := p.parseAwaitPrimary()
	if p.acceptOp("**") {
		return p.finish(&Node{Kind: BinOp, Op: "**", Left: base, Values: one(p.parseFactor())}, start)
	}
	return base
}

func (p *Parser) parseAwaitPrimary() *Node {
	if p.atKeyword("await") {
		start := p.mark()
		p.next()
		return p.finish(&Node{Kind: Await, Values: one(p.parsePrimary())}, start)
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() *Node {
	start := p.mark()
	node := p.parseAtom()
	for {
		switch {
		case p.atOp("."):
			p.next()
			node = p.finish(&Node{Kind: Attribute, Object: node, Name: p.expectName()}, start)
		case p.atOp("("):
			p.next()
			call := &Node{Kind: Call, Func: node, Values: p.parseCallArguments()}
			p.expectOp(")")
			node = p.finish(call, start)
		case p.atOp("["):
			p.next()
			subscript := &Node{Kind: Subscript, Object: node, Values: p.parseSubscriptList()}
			p.expectOp("]")
			node = p.finish(subscript, start)
		default:
			return node
		}
	}
}

func (p *Parser) parseAtom() *Node {
	start := p.mark()
	token := p.peek()
	switch token.Type {
	case Name:
		switch token.Text {
		case "None", "True", "False":
			p.next()
			return p.finish(&Node{Kind: Constant, Value: token.Text}, start)
		case "lambda":
			return p.parseLambda()
		case "yield":
			return p.parseYield()
		case "await":
			return p.parseAwaitPrimary()
		case "not":
			return p.parseInversion()
		}
		if nonStartKeywords[token.Text] {
			p.fail("unexpected keyword")
		}
		p.next()
		return p.finish(&Node{Kind: NameNode, Name: token.Text}, start)
	case Number:
		p.next()
		return p.finish(&Node{Kind: Constant, Value: token.Text}, start)
	case String:
		p.next()
		for p.at(String) {
			p.next()
		}
		return p.finish(&Node{Kind: Constant, Value: token.Text}, start)
	case Op:
		switch token.Text {
		case "(":
			return p.parseParenthesized()
		case "[":
			return p.parseList()
		case "{":
			return p.parseDictOrSet()
		case "...":
			p.next()
			return p.finish(&Node{Kind: Constant, Value: "..."}, start)
		}
	}
	p.fail("unexpected token")
	return nil
}

func (p *Parser) parseParenthesized() *Node {
	start := p.mark()
	p.expectOp("(")
	if p.acceptOp(")") {
		return p.finish(&Node{Kind: Tuple}, start)
	}
	if p.atKeyword("yield") {
		node := p.parseYield()
		p.expectOp(")")
		return node
	}
	first := p.parseElement()
	if p.atComprehensionStart() {
		node := &Node{Kind: GeneratorExp, Body: one(first), Generators: p.parseComprehensionClauses()}
		p.expectOp(")")
		return p.finish(node, start)
	}
	if p.atOp(",") {
		tuple := &Node{Kind: Tuple, Values: []*Node{first}}
		for p.acceptOp(",") {
			if p.atOp(")") {
				break
			}
			tuple.Values = append(tuple.Values, p.parseElement())
		}
		p.expectOp(")")
		return p.finish(tuple, start)
	}
	p.expectOp(")")
	return first
}

func (p *Parser) parseList() *Node {
	start := p.mark()
	p.expectOp("[")
	node := &Node{Kind: List}
	if p.acceptOp("]") {
		return p.finish(node, start)
	}
	first := p.parseElement()
	if p.atComprehensionStart() {
		node.Kind = ListComp
		node.Body = one(first)
		node.Generators = p.parseComprehensionClauses()
		p.expectOp("]")
		return p.finish(node, start)
	}
	node.Values = []*Node{first}
	for p.acceptOp(",") {
		if p.atOp("]") {
			break
		}
		node.Values = append(node.Values, p.parseElement())
	}
	p.expectOp("]")
	return p.finish(node, start)
}

func (p *Parser) parseDictOrSet() *Node {
	start := p.mark()
	p.expectOp("{")
	if p.acceptOp("}") {
		return p.finish(&Node{Kind: Dict}, start)
	}
	if p.atOp("**") {
		node := &Node{Kind: Dict}
		p.parseDictEntries(node)
		p.expectOp("}")
		return p.finish(node, start)
	}
	first := p.parseElement()
	if p.acceptOp(":") {
		value := p.parseElement()
		if p.atComprehensionStart() {
			node := &Node{Kind: DictComp, Body: []*Node{first, value}, Generators: p.parseComprehensionClauses()}
			p.expectOp("}")
			return p.finish(node, start)
		}
		node := &Node{Kind: Dict, Values: []*Node{first, value}}
		if p.acceptOp(",") {
			p.parseDictEntries(node)
		}
		p.expectOp("}")
		return p.finish(node, start)
	}
	if p.atComprehensionStart() {
		node := &Node{Kind: SetComp, Body: one(first), Generators: p.parseComprehensionClauses()}
		p.expectOp("}")
		return p.finish(node, start)
	}
	node := &Node{Kind: Set, Values: []*Node{first}}
	for p.acceptOp(",") {
		if p.atOp("}") {
			break
		}
		node.Values = append(node.Values, p.parseElement())
	}
	p.expectOp("}")
	return p.finish(node, start)
}

func (p *Parser) parseDictEntries(node *Node) {
	for !p.atOp("}") {
		if p.acceptOp("**") {
			node.Values = append(node.Values, p.parseBitOr())
		} else {
			key := p.parseElement()
			p.expectOp(":")
			node.Values = append(node.Values, key, p.parseElement())
		}
		if !p.acceptOp(",") {
			break
		}
	}
}

// parseElement parses one container element or argument; in pattern mode an
// 'as' binding may follow.
func (p *Parser) parseElement() *Node {
	node := p.parseStarExpression()
	if p.patternMode && p.acceptKeyword("as") {
		p.expectName()
	}
	return node
}

func (p *Parser) atComprehensionStart() bool {
	return p.atKeyword("for") || (p.atKeyword("async") && p.peekAt(1).Type == Name && p.peekAt(1).Text == "for")
}

func (p *Parser) parseComprehensionClauses() []*Node {
	var clauses []*Node
	for p.atComprehensionStart() {
		start := p.mark()
		clause := &Node{Kind: Comprehension}
		if p.acceptKeyword("async") {
			clause.Async = true
		}
		p.expectKeyword("for")
		clause.Target = p.parseTargetList()
		p.expectKeyword("in")
		clause.Iter = p.parseDisjunction()
		for p.atKeyword("if") {
			clause.OpLines = append(clause.OpLines, p.next().Line)
			clause.Values = append(clause.Values, p.parseDisjunction())
		}
		clauses = append(clauses, p.finish(clause, start))
	}
	return clauses
}

func (p *Parser) parseCallArguments() []*Node {
	var arguments []*Node
	for !p.atOp(")") {
		start := p.mark()
		switch {
		case p.atOp("**"):
			p.next()
			arguments = append(arguments, p.finish(&Node{Kind: Keyword, Values: one(p.parseExpression())}, start))
		case p.atOp("*"):
			p.next()
			arguments = append(arguments, p.finish(&Node{Kind: Starred, Values: one(p.parseExpression())}, start))
		case p.at(Name) && p.peekAt(1).Type == Op && p.peekAt(1).Text == "=":
			name := p.next().Text
			p.next()
			keyword := &Node{Kind: Keyword, Name: name, Values: one(p.parseExpression())}
			if p.patternMode && p.acceptKeyword("as") {
				p.expectName()
			}
			arguments = append(arguments, p.finish(keyword, start))
		default:
			argument := p.parseNamedExpr()
			if len(arguments) == 0 && p.atComprehensionStart() {
				argument = p.finish(&Node{Kind: GeneratorExp, Body: one(argument), Generators: p.parseComprehensionClauses()}, start)
			}
			if p.patternMode && p.acceptKeyword("as") {
				p.expectName()
			}
			arguments = append(arguments, argument)
		}
		if !p.acceptOp(",") {
			break
		}
	}
	return arguments
}

func (p *Parser) parseSubscriptList() []*Node {
	var items []*Node
	for {
		items = append(items, p.parseSliceOrExpression())
		if !p.acceptOp(",") {
			break
		}
		if p.atOp("]") {
			break
		}
	}
	return items
}

func (p *Parser) parseSliceOrExpression() *Node {
	start := p.mark()
	var lower *Node
	if !p.atOp(":") {
		lower = p.parseStarExpression()
		if !p.atOp(":") {
			return lower
		}
	}
	slice := &Node{Kind: Slice}
	if lower != nil {
		slice.Values = append(slice.Values, lower)
	}
	p.expectOp(":")
	if !p.atOp(":") && !p.atOp("]") && !p.atOp(",") {
		slice.Values = append(slice.Values, p.parseExpression())
	}
	if p.acceptOp(":") {
		if !p.atOp("]") && !p.atOp(",") {
			slice.Values = append(slice.Values, p.parseExpression())
		}
	}
	return p.finish(slice, start)
}

func (p *Parser) parseYield() *Node {
	start := p.mark()
	p.expectKeyword("yield")
	if p.acceptKeyword("from") {
		return p.finish(&Node{Kind: YieldFrom, Values: one(p.parseExpression())}, start)
	}
	node := &Node{Kind: Yield}
	if p.canStartExpression() {
		node.Values = one(p.parseStarExpressions())
	}
	return p.finish(node, start)
}
