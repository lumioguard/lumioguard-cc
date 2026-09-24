package cfamily

import (
	"fmt"

	"github.com/lumioguard/lumioguard-cc/internal/adapter/cfamily/syntax"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
)

// parser finds function definitions in a token stream and builds their
// control-flow trees. It reads declarations only as far as it must to find them.
type parser struct {
	tokens []syntax.Token
	// match holds the index of the partner of every bracket, and -1 elsewhere.
	match []int
	// cpp enables the alternative spellings "and" and "or" of && and ||.
	cpp       bool
	functions []*structure.Function
}

// scope is where a construct is declared.
type scope struct {
	// className is the nearest enclosing class, struct or union.
	className string
	// function is the raw name of the nearest enclosing function.
	function string
	// inClass is true directly inside a class body, where access labels occur.
	inClass bool
}

type parseError struct {
	line    int
	message string
}

func (e *parseError) Error() string {
	return fmt.Sprintf("%s (line %d)", e.message, e.line)
}

// parse returns every function definition of the file.
func parse(tokens []syntax.Token, cpp bool) (functions []*structure.Function, failure *parseError) {
	p := &parser{tokens: tokens, cpp: cpp}
	defer func() {
		if recovered := recover(); recovered != nil {
			failed, ok := recovered.(*parseError)
			if !ok {
				panic(recovered)
			}
			functions, failure = nil, failed
		}
	}()
	p.matchBrackets()
	p.declarations(0, len(tokens), scope{})
	return p.functions, nil
}

func (p *parser) fail(index int, format string, args ...any) {
	line := 1
	if len(p.tokens) > 0 {
		line = p.tokens[min(max(index, 0), len(p.tokens)-1)].Line
	}
	panic(&parseError{line: line, message: fmt.Sprintf(format, args...)})
}

var closers = map[string]string{"(": ")", "[": "]", "{": "}"}

// matchBrackets pairs every bracket, so later passes can skip a group in one step.
func (p *parser) matchBrackets() {
	p.match = make([]int, len(p.tokens))
	var open []int
	for index, token := range p.tokens {
		p.match[index] = -1
		if token.Kind != syntax.Punct {
			continue
		}
		switch token.Text {
		case "(", "[", "{":
			open = append(open, index)
		case ")", "]", "}":
			if len(open) == 0 {
				p.fail(index, "unmatched %q", token.Text)
			}
			opener := open[len(open)-1]
			open = open[:len(open)-1]
			if closers[p.tokens[opener].Text] != token.Text {
				p.fail(index, "%q closes %q opened on line %d", token.Text, p.tokens[opener].Text, p.tokens[opener].Line)
			}
			p.match[opener], p.match[index] = index, opener
		case "@":
			p.fail(index, "Objective-C syntax is not supported")
		}
	}
	if len(open) > 0 {
		p.fail(open[len(open)-1], "%q is never closed", p.tokens[open[len(open)-1]].Text)
	}
}

// is reports whether token i exists and is the punctuator or identifier text.
func (p *parser) is(i int, text string) bool {
	return i >= 0 && i < len(p.tokens) && p.tokens[i].Is(text)
}

func (p *parser) isOpen(i int) bool {
	return p.is(i, "(") || p.is(i, "[") || p.is(i, "{")
}

// isName reports whether token i is an identifier that can name a function.
func (p *parser) isName(i int) bool {
	if i < 0 || i >= len(p.tokens) || p.tokens[i].Kind != syntax.Identifier {
		return false
	}
	text := p.tokens[i].Text
	return !reserved[text] || !p.cpp && cppOnly[text]
}

// next skips token i, or its whole group when it opens one.
func (p *parser) next(i int) int {
	if p.isOpen(i) {
		return p.match[i] + 1
	}
	return i + 1
}

// semicolon returns the index of the first top-level ";" in [from, to), or to.
func (p *parser) semicolon(from, to int) int {
	for i := from; i < to; i = p.next(i) {
		if p.is(i, ";") {
			return i
		}
	}
	return to
}

// isIdentifier reports an identifier at i; it is false past either end of the file.
func (p *parser) isIdentifier(i int) bool {
	return i >= 0 && i < len(p.tokens) && p.tokens[i].Kind == syntax.Identifier
}

// isClassKey reports "class", "struct" or "union" at i.
func (p *parser) isClassKey(i int) bool {
	return p.tokens[i].Kind == syntax.Identifier && classKeys[p.tokens[i].Text]
}

// skipAngles returns the index after the template argument list opening at i.
func (p *parser) skipAngles(i int) int {
	depth := 0
	for ; i < len(p.tokens); i = p.next(i) {
		switch {
		case p.is(i, "<"):
			depth++
		case p.is(i, ">"):
			depth--
		case p.is(i, ">>"):
			depth -= 2
		case p.is(i, ";") || p.is(i, "{"):
			return i
		}
		if depth <= 0 {
			return i + 1
		}
	}
	return i
}
