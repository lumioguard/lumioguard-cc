package cfamily

import (
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/adapter/cfamily/syntax"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
)

// head is a function declarator followed by a body.
type head struct {
	// start is the first token of the declaration, after any macro invocations.
	start     int
	name      string
	qualifier string
	// nameStart is the first token of the qualified name.
	nameStart int
	// params is the index of the "(" opening the parameter list.
	params int
	// body is the index of the "{" opening the body.
	body int
	// try is the index of "try" in a function-try-block, or -1.
	try int
}

// functionHead finds the declarator of a function whose body starts at or after open. The
// last name with a parameter list and a valid trailer wins, so leading macros are not names.
func (p *parser) functionHead(start, open, to int, className string) (head, bool) {
	candidates := p.candidates(start, open)
	for index := len(candidates) - 1; index >= 0; index-- {
		found := candidates[index]
		if p.followsDeclarator(start, found.nameStart, candidates[:index], className) {
			continue
		}
		if body, try, ok := p.trailer(p.match[found.params]+1, open, to); ok {
			found.body, found.try, found.start = body, try, start
			for _, before := range candidates[:index] {
				if closing := p.match[before.params]; closing < found.nameStart {
					found.start = closing + 1
				}
			}
			return found, true
		}
	}
	if found, ok := p.nestedDeclarator(start, open); ok {
		found.body, found.try, found.start = open, -1, start
		return found, true
	}
	return head{}, false
}

// candidates lists every top-level name directly followed by a parenthesised group.
func (p *parser) candidates(from, to int) []head {
	var out []head
	previous := -1
	for i := from; i < to; {
		next := p.next(i)
		switch {
		case p.is(i, ":") && len(out) > 0:
			// Constructor initializers follow; "member(value)" there is no declarator.
			return out
		case p.is(i, "template") && p.is(i+1, "<"):
			next = p.skipAngles(i + 1)
		case p.is(i, "operator"):
			if found, ok := p.operatorHead(i, from, to); ok {
				out = append(out, found)
				next = p.match[found.params] + 1
			}
		case p.is(i, "("):
			if found, ok := p.declaratorBefore(previous, i, from); ok {
				out = append(out, found)
			}
		}
		previous, i = next-1, next
	}
	return out
}

func (p *parser) declaratorBefore(previous, paren, from int) (head, bool) {
	if p.is(previous, ")") && p.match[previous] >= from {
		return p.wrappedName(previous, paren, from)
	}
	name := previous
	if p.is(name, ">") {
		name = p.templateNameBefore(name, from)
	}
	if name < from || !p.isName(name) {
		return head{}, false
	}
	nameStart, qualifier := p.qualifiedStart(name, from)
	text := p.tokens[name].Text
	if p.is(name-1, "~") && name-1 >= from {
		text = "~" + text
	}
	return head{name: text, qualifier: qualifier, nameStart: nameStart, params: paren, try: -1}, true
}

// wrappedName reads "(min)()", which dodges a macro, and "C_SYMBOL(Yield)()", where a macro
// builds the name; a macro whose argument is not one name gives its own name.
func (p *parser) wrappedName(closing, paren, from int) (head, bool) {
	open := p.match[closing]
	name := closing - 1
	nameStart, qualifier := p.qualifiedStart(name, open+1)
	if p.isName(name) && nameStart == open+1 {
		return head{name: p.tokens[name].Text, qualifier: qualifier, nameStart: open, params: paren, try: -1}, true
	}
	if open-1 >= from && p.isName(open-1) {
		return head{name: p.tokens[open-1].Text, nameStart: open - 1, params: paren, try: -1}, true
	}
	return head{}, false
}

// operatorHead reads "operator" and its spelling up to the parameter list.
func (p *parser) operatorHead(keyword, from, to int) (head, bool) {
	end := keyword + 1
	if p.is(end, "(") && p.is(end+1, ")") {
		end += 2
	}
	for end < to && !p.is(end, "(") {
		end = p.next(end)
	}
	if end >= to {
		return head{}, false
	}
	nameStart, qualifier := p.qualifiedStart(keyword, from)
	name := "operator" + p.spelling(keyword+1, end)
	return head{name: name, qualifier: qualifier, nameStart: nameStart, params: end, try: -1}, true
}

// spelling joins tokens as written, with a space only between words.
func (p *parser) spelling(from, to int) string {
	var out strings.Builder
	for i := from; i < to; i++ {
		text := p.tokens[i].Text
		if p.tokens[i].Kind == syntax.Identifier && (out.Len() == 0 || isWordByte(out.String()[out.Len()-1])) {
			out.WriteByte(' ')
		}
		out.WriteString(text)
	}
	return out.String()
}

func isWordByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// templateNameBefore returns the name before the template arguments ending at
// closing, or -1.
func (p *parser) templateNameBefore(closing, from int) int {
	depth := 0
	for j := closing; j >= from; j-- {
		switch {
		case p.is(j, ">"):
			depth++
		case p.is(j, ">>"):
			depth += 2
		case p.is(j, "<"):
			depth--
		case p.is(j, ")") || p.is(j, "]") || p.is(j, "}"):
			j = p.match[j]
		}
		if depth == 0 {
			return j - 1
		}
	}
	return -1
}

// qualifiedStart walks back over "Outer::Inner::" and returns where the
// qualified name starts and the qualifier without template arguments.
func (p *parser) qualifiedStart(name, from int) (int, string) {
	start := name
	if p.is(start-1, "~") && start-1 >= from {
		start--
	}
	var parts []string
	for start-2 >= from && p.is(start-1, "::") {
		owner := start - 2
		if p.is(owner, ">") {
			owner = p.templateNameBefore(owner, from)
		}
		if owner < from || p.tokens[owner].Kind != syntax.Identifier {
			break
		}
		parts = append([]string{p.tokens[owner].Text}, parts...)
		start = owner
	}
	if start-1 >= from && p.is(start-1, "::") {
		start--
	}
	return start, strings.Join(parts, "::")
}

// followsDeclarator reports a name that comes after a plausible declarator's
// parameter list and qualifiers, such as an annotation macro with arguments.
func (p *parser) followsDeclarator(start, nameStart int, earlier []head, className string) bool {
	k := nameStart - 1
	for k >= start {
		switch {
		case qualifierWords[p.tokens[k].Text] && p.tokens[k].Kind != syntax.String:
			k--
		case p.is(k, ")") && (p.is(p.match[k]-1, "noexcept") || p.is(p.match[k]-1, "throw")):
			k = p.match[k] - 2
		default:
			for index, candidate := range earlier {
				if p.match[candidate.params] == k {
					return p.plausible(earlier, index, start, className)
				}
			}
			return false
		}
	}
	return false
}

// plausible reports a candidate that names a member or follows a type, not a macro
// invocation at the start of the declaration or right after another's arguments.
func (p *parser) plausible(candidates []head, index, start int, className string) bool {
	candidate := candidates[index]
	if candidate.qualifier != "" || strings.HasPrefix(candidate.name, "~") || candidate.name == className {
		return true
	}
	if candidate.nameStart <= start {
		return false
	}
	for _, before := range candidates[:index] {
		if p.match[before.params] == candidate.nameStart-1 {
			return false
		}
	}
	return true
}

// trailer checks what follows a parameter list up to the body and returns
// the body's opening brace.
func (p *parser) trailer(j, open, to int) (body, try int, ok bool) {
	try = -1
	for j < to {
		switch {
		case j == open:
			return open, try, true
		case p.is(j, ":"):
			body, ok = p.constructorInitializers(j+1, to)
			return body, try, ok
		case p.is(j, "try"):
			try = j
			j++
		case p.is(j, "->") || p.is(j, "requires"):
			j = open
		default:
			if j, ok = p.annotation(j); !ok {
				return 0, 0, false
			}
		}
	}
	return 0, 0, false
}

// annotation steps over a qualifier, an attribute or an annotation macro
// after a parameter list, such as "const", "[[nodiscard]]" or "LOCKS(mu)".
func (p *parser) annotation(j int) (int, bool) {
	token := p.tokens[j]
	switch {
	case token.Kind != syntax.String && qualifierWords[token.Text]:
		return j + 1, true
	case p.is(j, "["):
		return p.match[j] + 1, true
	case token.Kind == syntax.Identifier && (attributeWords[token.Text] || !reserved[token.Text]):
		if p.is(j+1, "(") {
			return p.match[j+1] + 1, true
		}
		return j + 1, true
	}
	return j, false
}

// constructorInitializers walks "a(1), b{2}, Base<T>(x)..." and returns the body brace.
func (p *parser) constructorInitializers(j, to int) (int, bool) {
	for j < to {
		for j < to && (p.isIdentifier(j) || p.is(j, "::")) {
			j++
			if p.is(j, "<") {
				j = p.skipAngles(j)
			}
		}
		if !p.is(j, "(") && !p.is(j, "{") {
			return 0, false
		}
		j = p.match[j] + 1
		if p.is(j, "...") {
			j++
		}
		switch {
		case p.is(j, "{"):
			return j, true
		case p.is(j, ","):
			j++
		case p.isIdentifier(j):
			// A macro such as a warning pragma may stand between items without a comma.
		default:
			return 0, false
		}
	}
	return 0, false
}

// nestedDeclarator finds the name in a declarator such as
// "void (*signal(int, handler))(int)", a function returning a function pointer.
func (p *parser) nestedDeclarator(start, open int) (head, bool) {
	for i := start; i < open; i = p.next(i) {
		if !p.is(i, "(") {
			continue
		}
		inner := i + 1
		if !p.is(inner, "*") && !p.is(inner, "&") && !p.is(inner, "&&") && !p.is(inner, "^") {
			return head{}, false
		}
		if found := p.candidates(inner, p.match[i]); len(found) > 0 {
			return found[len(found)-1], true
		}
		return head{}, false
	}
	return head{}, false
}

// defineFunction records the function whose declaration starts at found.start.
func (p *parser) defineFunction(found head, where scope) (int, []*structure.Node) {
	function := &structure.Function{
		Name:          found.name,
		ClassName:     where.className,
		EnclosingName: where.function,
		Line:          p.tokens[found.start].Line,
		Start:         p.tokens[found.start].Start,
		Parameters:    p.parameters(found.params),
	}
	if found.qualifier != "" {
		function.ClassName = found.qualifier
	}
	inner := scope{className: function.ClassName, function: function.Name}
	body := p.block(found.body, inner)
	end := p.match[found.body]
	if found.try >= 0 {
		handlers, last := p.handlers(end+1, len(p.tokens), inner)
		if len(handlers) == 0 {
			p.fail(end+1, "function-try-block without a handler")
		}
		body = structure.One(&structure.Node{
			Kind: structure.KindTry, Label: "TryStatement", Line: p.tokens[found.try].Line, Body: body, Handlers: handlers,
		})
		end = last
	}
	p.finish(function, body, end)
	return end + 1, structure.One(&structure.Node{
		Kind: structure.KindFunction, Label: "FunctionDefinition", Line: function.Line, Body: body,
	})
}

// finish completes a function ending at token end and records it.
func (p *parser) finish(function *structure.Function, body []*structure.Node, end int) {
	function.Body = &structure.Node{Kind: structure.KindBlock, Children: body}
	function.EndLine = p.tokens[end].EndLine
	function.End = p.tokens[end].End
	p.functions = append(p.functions, function)
}
