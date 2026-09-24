package cfamily

import (
	"github.com/lumioguard/lumioguard-cc/internal/adapter/cfamily/syntax"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
)

// declarations reads the declarations in [from, to) and returns a function
// node for each function defined there, for local classes inside a function.
func (p *parser) declarations(from, to int, where scope) []*structure.Node {
	var nodes []*structure.Node
	for i := from; i < to; {
		next, defined := p.declaration(i, to, where)
		nodes = append(nodes, defined...)
		i = next
	}
	return nodes
}

// declaration reads one declaration starting at start and returns the index
// after it. Only a brace or "=" at the top level needs a closer look.
func (p *parser) declaration(start, to int, where scope) (int, []*structure.Node) {
	if where.inClass {
		if next := p.accessLabel(start, to); next > start {
			return next, nil
		}
	}
	for i := start; i < to; i = p.declarationStep(i) {
		switch {
		case p.is(i, ";"):
			return i + 1, nil
		case p.is(i, "{"):
			return p.braced(start, i, to, where)
		case p.is(i, "="):
			return p.initializer(i, to, where), nil
		case where.inClass && i > start && p.accessLabel(i, to) > i:
			return i, nil
		}
	}
	return to, nil
}

// declarationStep skips one token or group of a declaration, including a
// template parameter list and an operator name, whose spelling may hold "=".
func (p *parser) declarationStep(i int) int {
	switch {
	case p.is(i, "template") && p.is(i+1, "<"):
		return p.skipAngles(i + 1)
	case p.is(i, "operator"):
		return p.skipOperatorName(i)
	}
	return p.next(i)
}

// accessLabel returns the index after an access label such as "public:" or
// Qt's "public slots:" at i, or i when there is none.
func (p *parser) accessLabel(i, to int) int {
	switch {
	case !p.isWord(i, accessWords):
	case p.is(i+1, ":"):
		return i + 2
	case accessSpecifiers[p.tokens[i].Text] && i+2 < to && p.tokens[i+1].Kind == syntax.Identifier && p.is(i+2, ":"):
		return i + 3
	}
	return i
}

// isWord reports an identifier at i that is one of words.
func (p *parser) isWord(i int, words map[string]bool) bool {
	return i < len(p.tokens) && p.tokens[i].Kind == syntax.Identifier && words[p.tokens[i].Text]
}

// skipOperatorName steps over "operator" and the operator it names, whose
// spelling may contain "=" or brackets.
func (p *parser) skipOperatorName(i int) int {
	if p.is(i+1, "(") || p.is(i+1, "[") {
		return p.match[i+1] + 1
	}
	return i + 2
}

// initializer reads "= value;" and registers any lambdas in the value.
func (p *parser) initializer(equals, to int, where scope) int {
	end := p.semicolon(equals+1, to)
	p.value(equals+1, end, p.nameBefore(equals), where)
	return min(end+1, to)
}

// braced decides what the brace at open belongs to.
func (p *parser) braced(start, open, to int, where scope) (int, []*structure.Node) {
	closing := p.match[open]
	if p.isNamespace(start, open) {
		return closing + 1, p.declarations(open+1, closing, scope{function: where.function})
	}
	if head, ok := p.functionHead(start, open, to, where.className); ok {
		return p.defineFunction(head, where)
	}
	if start == open {
		p.fail(open, "block outside a function; K&R-style definitions are not supported")
	}
	if name, ok := p.classHead(start, open); ok {
		return closing + 1, p.declarations(open+1, closing, scope{className: name, function: where.function, inClass: true})
	}
	// Skipping declarations as if they were an initializer would drop every
	// function in them without a trace, so an unrecognised body fails the file.
	if p.holdsDeclarations(open) {
		p.fail(open, "unrecognised declaration from line %d before '{'; a macro without a semicolon may hide it",
			p.tokens[start].Line)
	}
	// An enum body or a brace initializer: only lambdas inside matter.
	p.expression(open+1, closing, where)
	return closing + 1, nil
}

// holdsDeclarations reports a brace group that cannot be an initializer: it
// has a top-level ";" or a function body, a brace after ")" that no lambda owns.
func (p *parser) holdsDeclarations(open int) bool {
	closing := p.match[open]
	for i := open + 1; i < closing; i = p.next(i) {
		if p.is(i, "[") {
			if _, end, ok := p.lambdaParts(i, closing); ok {
				i = end
				continue
			}
		}
		if p.is(i, ";") || p.is(i, "{") && p.is(i-1, ")") {
			return true
		}
	}
	return false
}

// isNamespace reports a namespace body or a linkage block such as extern "C" {.
// A macro without a semicolon, such as ABSL_NAMESPACE_BEGIN, may come first.
func (p *parser) isNamespace(start, open int) bool {
	if open-2 >= start && p.is(open-2, "extern") && p.tokens[open-1].Kind == syntax.String {
		return true
	}
	for i := open - 1; i >= start; i-- {
		switch {
		case p.is(i, "namespace"):
			return true
		case p.is(i, "]"):
			i = p.match[i]
		case p.tokens[i].Kind != syntax.Identifier && !p.is(i, "::"):
			return false
		}
	}
	return false
}

// classHead returns the name of the class, struct or union whose body opens at
// open, "<anonymous>" for an unnamed one, and false for anything else.
func (p *parser) classHead(start, open int) (string, bool) {
	key := -1
	for i := start; i < open; i = p.declarationStep(i) {
		if p.is(i, "enum") {
			return "", false
		}
		if key < 0 && p.isClassKey(i) {
			key = i
		}
	}
	if key < 0 {
		return "", false
	}
	return p.className(key+1, open), true
}

// className returns the last name between a class key and the base list or
// body, which skips export macros and attributes before the name.
func (p *parser) className(from, open int) string {
	name := "<anonymous>"
	for i := from; i < open && !p.is(i, ":") && !p.is(i, "final"); i = p.next(i) {
		switch {
		case p.is(i, "<"):
			i = p.skipAngles(i) - 1
		case p.isName(i):
			name = p.tokens[i].Text
		}
	}
	return name
}

// nameBefore returns the identifier just before index i, or "".
func (p *parser) nameBefore(i int) string {
	if p.isName(i - 1) {
		return p.tokens[i-1].Text
	}
	return ""
}
