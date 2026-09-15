package syntax

import "fmt"

// Parser is a recursive-descent parser over the token stream.
type Parser struct {
	tokens      []Token
	pos         int
	last        int
	patternMode bool
}

type parseError struct {
	err *Error
}

// Parse tokenizes and parses Python 3 source. It returns the module node, the
// tokens, the comment ranges, or the first syntax error.
func Parse(src string) (module *Node, tokens []Token, comments []Comment, err error) {
	tokens, comments, err = Tokenize(src)
	if err != nil {
		return nil, nil, nil, err
	}
	p := &Parser{tokens: tokens, last: -1}
	defer func() {
		if recovered := recover(); recovered != nil {
			failure, ok := recovered.(parseError)
			if !ok {
				panic(recovered)
			}
			module, err = nil, failure.err
		}
	}()
	module = p.parseModule()
	return module, tokens, comments, nil
}

var nonStartKeywords = map[string]bool{
	"and": true, "or": true, "in": true, "is": true, "if": true, "else": true, "elif": true,
	"for": true, "while": true, "with": true, "as": true, "from": true, "import": true,
	"def": true, "class": true, "return": true, "try": true, "except": true, "finally": true,
	"raise": true, "pass": true, "break": true, "continue": true, "global": true,
	"nonlocal": true, "del": true, "assert": true,
}

var augmentedAssignments = map[string]bool{
	"+=": true, "-=": true, "*=": true, "/=": true, "//=": true, "%=": true, "@=": true,
	"&=": true, "|=": true, "^=": true, ">>=": true, "<<=": true, "**=": true,
}

var comparisonOperators = map[string]bool{"<": true, ">": true, "==": true, ">=": true, "<=": true, "!=": true}

// --- token helpers ---

func (p *Parser) peek() Token {
	return p.tokens[p.pos]
}

func (p *Parser) peekAt(offset int) Token {
	index := p.pos + offset
	if index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[index]
}

func (p *Parser) next() Token {
	token := p.tokens[p.pos]
	if token.Type != EOF {
		p.pos++
	}
	if token.Type == Name || token.Type == Number || token.Type == String || token.Type == Op {
		p.last = p.pos - 1
	}
	return token
}

func (p *Parser) at(tokenType TokenType) bool {
	return p.peek().Type == tokenType
}

func (p *Parser) atOp(text string) bool {
	token := p.peek()
	return token.Type == Op && token.Text == text
}

func (p *Parser) atKeyword(text string) bool {
	token := p.peek()
	return token.Type == Name && token.Text == text
}

func (p *Parser) acceptOp(text string) bool {
	if p.atOp(text) {
		p.next()
		return true
	}
	return false
}

func (p *Parser) acceptKeyword(text string) bool {
	if p.atKeyword(text) {
		p.next()
		return true
	}
	return false
}

func (p *Parser) acceptNewline() bool {
	if p.at(Newline) {
		p.next()
		return true
	}
	return false
}

func (p *Parser) expectOp(text string) {
	if !p.atOp(text) {
		p.fail("expected '%s'", text)
	}
	p.next()
}

func (p *Parser) expectKeyword(text string) {
	if !p.atKeyword(text) {
		p.fail("expected '%s'", text)
	}
	p.next()
}

func (p *Parser) expect(tokenType TokenType) {
	if !p.at(tokenType) {
		p.fail("expected %s", tokenType)
	}
	p.next()
}

func (p *Parser) expectName() string {
	token := p.peek()
	if token.Type != Name || nonStartKeywords[token.Text] {
		p.fail("expected a name")
	}
	p.next()
	return token.Text
}

func (p *Parser) fail(format string, args ...any) {
	token := p.peek()
	message := fmt.Sprintf(format, args...)
	if token.Type == EOF {
		message += " at end of file"
	} else if token.Text != "" {
		message += fmt.Sprintf(" near %q", token.Text)
	}
	panic(parseError{err: &Error{Line: token.Line, Message: message}})
}

// attempt runs parse and rolls back on a syntax error.
func (p *Parser) attempt(parse func()) (ok bool) {
	pos, last, mode := p.pos, p.last, p.patternMode
	defer func() {
		if recovered := recover(); recovered != nil {
			if _, isParse := recovered.(parseError); !isParse {
				panic(recovered)
			}
			p.pos, p.last, p.patternMode = pos, last, mode
			ok = false
		}
	}()
	parse()
	return true
}

func (p *Parser) mark() int {
	return p.pos
}

func (p *Parser) finish(node *Node, startIndex int) *Node {
	start := p.tokens[startIndex]
	node.Start, node.Line = start.Start, start.Line
	if p.last >= startIndex {
		end := p.tokens[p.last]
		node.End, node.EndLine = end.End, end.EndLine
	} else {
		node.End, node.EndLine = start.End, start.EndLine
	}
	return node
}

func (p *Parser) atStatementEnd() bool {
	token := p.peek()
	return token.Type == Newline || token.Type == EOF || (token.Type == Op && token.Text == ";")
}

func (p *Parser) canStartExpression() bool {
	token := p.peek()
	switch token.Type {
	case Name:
		return !nonStartKeywords[token.Text]
	case Number, String:
		return true
	case Op:
		switch token.Text {
		case "(", "[", "{", "-", "+", "~", "*", "**", "...":
			return true
		}
	}
	return false
}

// --- module and statements ---

func (p *Parser) parseModule() *Node {
	module := &Node{Kind: ModuleNode, Line: 1, EndLine: 1}
	for !p.at(EOF) {
		if p.acceptNewline() {
			continue
		}
		module.Body = append(module.Body, p.parseStatement()...)
	}
	if len(p.tokens) > 0 {
		p.finish(module, 0)
	}
	return module
}

func (p *Parser) parseStatement() []*Node {
	token := p.peek()
	if token.Type == Name {
		switch token.Text {
		case "if":
			return one(p.parseIfChain("if"))
		case "while":
			return one(p.parseWhile())
		case "for":
			return one(p.parseFor(p.mark(), false))
		case "try":
			return one(p.parseTry())
		case "with":
			return one(p.parseWith(p.mark(), false))
		case "def":
			return one(p.parseDef(p.mark(), false, nil))
		case "class":
			return one(p.parseClass(p.mark(), nil))
		case "async":
			start := p.mark()
			switch p.peekAt(1).Text {
			case "def":
				p.next()
				return one(p.parseDef(start, true, nil))
			case "for":
				p.next()
				return one(p.parseFor(start, true))
			case "with":
				p.next()
				return one(p.parseWith(start, true))
			}
		case "match":
			if node, ok := p.tryMatch(); ok {
				return one(node)
			}
		}
	}
	if token.Type == Op && token.Text == "@" {
		return one(p.parseDecorated())
	}
	return p.parseSimpleStatements()
}

func one(node *Node) []*Node {
	return []*Node{node}
}

func (p *Parser) parseSimpleStatements() []*Node {
	var statements []*Node
	for {
		statements = append(statements, p.parseSimpleStatement())
		if p.acceptOp(";") {
			if p.at(Newline) || p.at(EOF) {
				break
			}
			continue
		}
		break
	}
	if !p.acceptNewline() && !p.at(EOF) {
		p.fail("expected end of statement")
	}
	return statements
}

func (p *Parser) parseBlock() []*Node {
	if p.acceptNewline() {
		p.expect(Indent)
		var statements []*Node
		for !p.at(Dedent) && !p.at(EOF) {
			if p.acceptNewline() {
				continue
			}
			statements = append(statements, p.parseStatement()...)
		}
		p.expect(Dedent)
		return statements
	}
	return p.parseSimpleStatements()
}

func (p *Parser) parseSimpleStatement() *Node {
	start := p.mark()
	token := p.peek()
	if token.Type == Name {
		switch token.Text {
		case "pass":
			p.next()
			return p.finish(&Node{Kind: Pass}, start)
		case "break":
			p.next()
			return p.finish(&Node{Kind: Break}, start)
		case "continue":
			p.next()
			return p.finish(&Node{Kind: Continue}, start)
		case "return":
			p.next()
			node := &Node{Kind: Return}
			if !p.atStatementEnd() {
				node.Values = one(p.parseStarExpressions())
			}
			return p.finish(node, start)
		case "raise":
			p.next()
			node := &Node{Kind: Raise}
			if !p.atStatementEnd() {
				node.Values = one(p.parseExpression())
				if p.acceptKeyword("from") {
					node.Values = append(node.Values, p.parseExpression())
				}
			}
			return p.finish(node, start)
		case "global", "nonlocal":
			p.next()
			kind := Global
			if token.Text == "nonlocal" {
				kind = Nonlocal
			}
			node := &Node{Kind: kind}
			for {
				node.Names = append(node.Names, Alias{Name: p.expectName()})
				if !p.acceptOp(",") {
					break
				}
			}
			return p.finish(node, start)
		case "del":
			p.next()
			return p.finish(&Node{Kind: Delete, Values: one(p.parseStarExpressions())}, start)
		case "assert":
			p.next()
			node := &Node{Kind: Assert, Test: p.parseExpression()}
			if p.acceptOp(",") {
				node.Values = one(p.parseExpression())
			}
			return p.finish(node, start)
		case "import":
			return p.parseImport()
		case "from":
			return p.parseImportFrom()
		case "type":
			if p.peekAt(1).Type == Name && p.peekAt(2).Type == Op && (p.peekAt(2).Text == "=" || p.peekAt(2).Text == "[") {
				return p.parseTypeAlias()
			}
		}
	}
	expression := p.parseStarExpressions()
	switch {
	case p.atOp("="):
		node := &Node{Kind: Assign, Values: []*Node{expression}}
		for p.acceptOp("=") {
			node.Values = append(node.Values, p.parseYieldOrStarExpressions())
		}
		return p.finish(node, start)
	case p.peek().Type == Op && augmentedAssignments[p.peek().Text]:
		operator := p.next().Text
		return p.finish(&Node{Kind: AugAssign, Op: operator, Target: expression, Values: one(p.parseYieldOrStarExpressions())}, start)
	case p.atOp(":"):
		p.next()
		node := &Node{Kind: AnnAssign, Target: expression, Values: one(p.parseExpression())}
		if p.acceptOp("=") {
			node.Values = append(node.Values, p.parseYieldOrStarExpressions())
		}
		return p.finish(node, start)
	default:
		return p.finish(&Node{Kind: ExprStmt, Values: one(expression)}, start)
	}
}

func (p *Parser) parseYieldOrStarExpressions() *Node {
	if p.atKeyword("yield") {
		return p.parseYield()
	}
	return p.parseStarExpressions()
}

func (p *Parser) parseIfChain(keyword string) *Node {
	start := p.mark()
	p.expectKeyword(keyword)
	node := &Node{Kind: If, Elif: keyword == "elif", Test: p.parseNamedExpr()}
	p.expectOp(":")
	node.Body = p.parseBlock()
	switch {
	case p.atKeyword("elif"):
		node.ElseLine = p.peek().Line
		node.OrElse = one(p.parseIfChain("elif"))
	case p.atKeyword("else"):
		node.ElseLine = p.peek().Line
		p.next()
		p.expectOp(":")
		node.OrElse = p.parseBlock()
	}
	return p.finish(node, start)
}

func (p *Parser) parseWhile() *Node {
	start := p.mark()
	p.expectKeyword("while")
	node := &Node{Kind: While, Test: p.parseNamedExpr()}
	p.expectOp(":")
	node.Body = p.parseBlock()
	p.parseLoopElse(node)
	return p.finish(node, start)
}

func (p *Parser) parseFor(start int, async bool) *Node {
	p.expectKeyword("for")
	node := &Node{Kind: For, Async: async, Target: p.parseTargetList()}
	p.expectKeyword("in")
	node.Iter = p.parseStarExpressions()
	p.expectOp(":")
	node.Body = p.parseBlock()
	p.parseLoopElse(node)
	return p.finish(node, start)
}

func (p *Parser) parseLoopElse(node *Node) {
	if p.atKeyword("else") {
		node.ElseLine = p.peek().Line
		p.next()
		p.expectOp(":")
		node.OrElse = p.parseBlock()
	}
}

func (p *Parser) parseTry() *Node {
	start := p.mark()
	p.expectKeyword("try")
	p.expectOp(":")
	node := &Node{Kind: Try, Body: p.parseBlock()}
	for p.atKeyword("except") {
		handlerStart := p.mark()
		p.next()
		handler := &Node{Kind: ExceptHandler}
		p.acceptOp("*")
		if !p.atOp(":") {
			handler.Test = p.parseExpression()
			if p.acceptKeyword("as") {
				handler.Name = p.expectName()
			}
		}
		p.expectOp(":")
		handler.Body = p.parseBlock()
		node.Handlers = append(node.Handlers, p.finish(handler, handlerStart))
	}
	if p.atKeyword("else") {
		node.ElseLine = p.peek().Line
		p.next()
		p.expectOp(":")
		node.OrElse = p.parseBlock()
	}
	if p.acceptKeyword("finally") {
		p.expectOp(":")
		node.Final = p.parseBlock()
	}
	if len(node.Handlers) == 0 && node.Final == nil {
		p.fail("expected 'except' or 'finally' block")
	}
	return p.finish(node, start)
}

func (p *Parser) parseWith(start int, async bool) *Node {
	p.expectKeyword("with")
	node := &Node{Kind: With, Async: async}
	parenthesized := false
	if p.atOp("(") {
		parenthesized = p.attempt(func() {
			p.expectOp("(")
			var items []*Node
			for !p.atOp(")") {
				items = append(items, p.parseWithItem())
				if !p.acceptOp(",") {
					break
				}
			}
			p.expectOp(")")
			p.expectOp(":")
			node.Values = items
		})
	}
	if !parenthesized {
		for {
			node.Values = append(node.Values, p.parseWithItem())
			if !p.acceptOp(",") {
				break
			}
		}
		p.expectOp(":")
	}
	node.Body = p.parseBlock()
	return p.finish(node, start)
}

func (p *Parser) parseWithItem() *Node {
	item := p.parseExpression()
	if p.acceptKeyword("as") {
		p.parseTarget()
	}
	return item
}

func (p *Parser) parseDecorated() *Node {
	var decorators []*Node
	for p.acceptOp("@") {
		decorators = append(decorators, p.parseNamedExpr())
		p.expect(Newline)
	}
	start := p.mark()
	switch {
	case p.atKeyword("def"):
		return p.parseDef(start, false, decorators)
	case p.atKeyword("class"):
		return p.parseClass(start, decorators)
	case p.atKeyword("async") && p.peekAt(1).Text == "def":
		p.next()
		return p.parseDef(start, true, decorators)
	}
	p.fail("expected a function or class after decorators")
	return nil
}

func (p *Parser) parseDef(start int, async bool, decorators []*Node) *Node {
	p.expectKeyword("def")
	node := &Node{Kind: FunctionDef, Async: async, Decorators: decorators, Name: p.expectName()}
	if p.atOp("[") {
		p.parseTypeParams()
	}
	p.expectOp("(")
	node.Params, node.Values = p.parseParameters(")", true)
	p.expectOp(")")
	if p.acceptOp("->") {
		node.Values = append(node.Values, p.parseStarOrExpression())
	}
	p.expectOp(":")
	node.Body = p.parseBlock()
	return p.finish(node, start)
}

// parseParameters parses a parameter list up to the closing token and returns the
// slot count plus the default and annotation expressions.
func (p *Parser) parseParameters(closing string, annotations bool) (int, []*Node) {
	count := 0
	var expressions []*Node
	for !p.atOp(closing) {
		switch {
		case p.atOp("/"):
			p.next()
		case p.atOp("**"):
			p.next()
			p.expectName()
			if annotations && p.acceptOp(":") {
				expressions = append(expressions, p.parseExpression())
			}
			count++
		case p.atOp("*"):
			p.next()
			if p.at(Name) && !p.atKeyword("lambda") {
				p.expectName()
				if annotations && p.acceptOp(":") {
					expressions = append(expressions, p.parseStarOrExpression())
				}
				count++
			}
		default:
			p.expectName()
			if annotations && p.acceptOp(":") {
				expressions = append(expressions, p.parseExpression())
			}
			if p.acceptOp("=") {
				expressions = append(expressions, p.parseExpression())
			}
			count++
		}
		if !p.acceptOp(",") {
			break
		}
	}
	return count, expressions
}

func (p *Parser) parseTypeParams() {
	p.expectOp("[")
	for !p.atOp("]") {
		if !p.acceptOp("**") {
			p.acceptOp("*")
		}
		p.expectName()
		if p.acceptOp(":") {
			p.parseExpression()
		}
		if p.acceptOp("=") {
			p.parseExpression()
		}
		if !p.acceptOp(",") {
			break
		}
	}
	p.expectOp("]")
}

func (p *Parser) parseClass(start int, decorators []*Node) *Node {
	p.expectKeyword("class")
	node := &Node{Kind: ClassDef, Decorators: decorators, Name: p.expectName()}
	if p.atOp("[") {
		p.parseTypeParams()
	}
	if p.acceptOp("(") {
		node.Values = p.parseCallArguments()
		p.expectOp(")")
	}
	p.expectOp(":")
	node.Body = p.parseBlock()
	return p.finish(node, start)
}

func (p *Parser) tryMatch() (*Node, bool) {
	start := p.mark()
	var node *Node
	ok := p.attempt(func() {
		p.expectKeyword("match")
		subject := p.parseStarExpressions()
		p.expectOp(":")
		p.expect(Newline)
		p.expect(Indent)
		if !p.atKeyword("case") {
			p.fail("expected 'case'")
		}
		node = &Node{Kind: Match, Test: subject}
		for p.atKeyword("case") {
			node.Cases = append(node.Cases, p.parseCase())
		}
		for p.acceptNewline() {
		}
		p.expect(Dedent)
		p.finish(node, start)
	})
	return node, ok
}

func (p *Parser) parseCase() *Node {
	start := p.mark()
	p.expectKeyword("case")
	node := &Node{Kind: MatchCase}
	p.patternMode = true
	node.Pattern = p.parsePatternList()
	p.patternMode = false
	if p.acceptKeyword("if") {
		node.Guard = p.parseNamedExpr()
	}
	p.expectOp(":")
	node.Body = p.parseBlock()
	node.Wildcard = node.Guard == nil && node.Pattern.Kind == NameNode && node.Pattern.Name == "_"
	return p.finish(node, start)
}

func (p *Parser) parsePatternList() *Node {
	start := p.mark()
	first := p.parsePattern()
	if !p.atOp(",") {
		return first
	}
	tuple := &Node{Kind: Tuple, Values: []*Node{first}}
	for p.acceptOp(",") {
		if p.atOp(":") || p.atKeyword("if") {
			break
		}
		tuple.Values = append(tuple.Values, p.parsePattern())
	}
	return p.finish(tuple, start)
}

func (p *Parser) parsePattern() *Node {
	start := p.mark()
	pattern := p.parseStarOrBitOr()
	if p.acceptKeyword("as") {
		p.expectName()
		return p.finish(&Node{Kind: NamedExpr, Values: one(pattern)}, start)
	}
	return pattern
}

func (p *Parser) parseImport() *Node {
	start := p.mark()
	p.expectKeyword("import")
	node := &Node{Kind: Import}
	for {
		alias := Alias{Name: p.parseDottedName()}
		if p.acceptKeyword("as") {
			alias.AsName = p.expectName()
		}
		node.Names = append(node.Names, alias)
		if !p.acceptOp(",") {
			break
		}
	}
	return p.finish(node, start)
}

func (p *Parser) parseImportFrom() *Node {
	start := p.mark()
	p.expectKeyword("from")
	node := &Node{Kind: ImportFrom}
	for {
		if p.acceptOp(".") {
			node.Level++
			continue
		}
		if p.acceptOp("...") {
			node.Level += 3
			continue
		}
		break
	}
	if !p.atKeyword("import") {
		node.Module = p.parseDottedName()
	}
	p.expectKeyword("import")
	if p.acceptOp("*") {
		node.Names = []Alias{{Name: "*"}}
		return p.finish(node, start)
	}
	parenthesized := p.acceptOp("(")
	for {
		alias := Alias{Name: p.expectName()}
		if p.acceptKeyword("as") {
			alias.AsName = p.expectName()
		}
		node.Names = append(node.Names, alias)
		if !p.acceptOp(",") {
			break
		}
		if parenthesized && p.atOp(")") {
			break
		}
	}
	if parenthesized {
		p.expectOp(")")
	}
	return p.finish(node, start)
}

func (p *Parser) parseDottedName() string {
	name := p.expectName()
	for p.atOp(".") && p.peekAt(1).Type == Name {
		p.next()
		name += "." + p.expectName()
	}
	return name
}

func (p *Parser) parseTypeAlias() *Node {
	start := p.mark()
	p.expectKeyword("type")
	node := &Node{Kind: TypeAlias, Name: p.expectName()}
	if p.atOp("[") {
		p.parseTypeParams()
	}
	p.expectOp("=")
	node.Values = one(p.parseExpression())
	return p.finish(node, start)
}

// parseTargetList parses assignment targets of for loops and comprehensions,
// which stop before the 'in' keyword.
func (p *Parser) parseTargetList() *Node {
	start := p.mark()
	first := p.parseTarget()
	if !p.atOp(",") {
		return first
	}
	tuple := &Node{Kind: Tuple, Values: []*Node{first}}
	for p.acceptOp(",") {
		if p.atKeyword("in") || p.atOp("=") || p.atOp(")") || p.atOp("]") || p.atOp(":") {
			break
		}
		tuple.Values = append(tuple.Values, p.parseTarget())
	}
	return p.finish(tuple, start)
}

func (p *Parser) parseTarget() *Node {
	return p.parseStarOrBitOr()
}

func (p *Parser) parseStarOrBitOr() *Node {
	if p.atOp("*") {
		start := p.mark()
		p.next()
		return p.finish(&Node{Kind: Starred, Values: one(p.parseBitOr())}, start)
	}
	return p.parseBitOr()
}
