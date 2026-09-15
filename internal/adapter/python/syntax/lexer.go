package syntax

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var stringPrefixes = map[string]bool{
	"r": true, "u": true, "b": true, "f": true, "t": true,
	"br": true, "rb": true, "fr": true, "rf": true, "tr": true, "rt": true,
}

var operators = []string{
	"**=", "//=", ">>=", "<<=", "...",
	"**", "//", ">>", "<<", "<=", ">=", "==", "!=", "->", ":=",
	"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "@=",
	"+", "-", "*", "/", "%", "@", "&", "|", "^", "~", "<", ">", "=",
	".", ",", ":", ";", "(", ")", "[", "]", "{", "}", "!",
}

type lexError struct {
	err *Error
}

type lexer struct {
	src           string
	pos           int
	line          int
	tokens        []Token
	comments      []Comment
	indents       []int
	depth         int
	lineHasTokens bool
	atLineStart   bool
}

// Tokenize splits Python source into tokens and comment ranges.
func Tokenize(src string) (tokens []Token, comments []Comment, err error) {
	lx := &lexer{src: src, line: 1, indents: []int{0}, atLineStart: true}
	defer func() {
		if recovered := recover(); recovered != nil {
			failure, ok := recovered.(lexError)
			if !ok {
				panic(recovered)
			}
			tokens, comments, err = nil, nil, failure.err
		}
	}()
	lx.run()
	return lx.tokens, lx.comments, nil
}

func (lx *lexer) fail(format string, args ...any) {
	panic(lexError{err: &Error{Line: lx.line, Message: fmt.Sprintf(format, args...)}})
}

func (lx *lexer) emit(tokenType TokenType, start, end, line, endLine int) {
	text := ""
	if tokenType == Name || tokenType == Number || tokenType == String || tokenType == Op {
		text = lx.src[start:end]
		lx.lineHasTokens = true
	}
	lx.tokens = append(lx.tokens, Token{Type: tokenType, Text: text, Start: start, End: end, Line: line, EndLine: endLine})
}

func (lx *lexer) run() {
	if len(lx.src) >= 3 && lx.src[0] == 0xEF && lx.src[1] == 0xBB && lx.src[2] == 0xBF {
		lx.pos = 3
	}
	for lx.pos < len(lx.src) {
		if lx.atLineStart && lx.depth == 0 && lx.handleIndentation() {
			continue
		}
		if lx.pos >= len(lx.src) {
			break
		}
		c := lx.src[lx.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\f':
			lx.pos++
		case c == '\\' && lx.pos+1 < len(lx.src) && isNewline(lx.src[lx.pos+1]):
			lx.pos++
			lx.consumeNewline()
		case c == '\\' && lx.pos+2 < len(lx.src) && lx.src[lx.pos+1] == '\r' && lx.src[lx.pos+2] == '\n':
			lx.pos++
			lx.consumeNewline()
		case c == '#':
			lx.comment()
		case isNewline(c):
			lx.newline()
		case c == '\'' || c == '"':
			lx.scanString(lx.pos, "")
		case isDigit(c) || (c == '.' && lx.pos+1 < len(lx.src) && isDigit(lx.src[lx.pos+1])):
			lx.number()
		case isIdentifierStart(lx.src, lx.pos):
			lx.identifierOrString()
		default:
			lx.operator()
		}
	}
	// Unclosed brackets are left to the parser, which reports the offending
	// token instead of the end of the file.
	if lx.lineHasTokens {
		lx.emit(Newline, lx.pos, lx.pos, lx.line, lx.line)
		lx.lineHasTokens = false
	}
	for len(lx.indents) > 1 {
		lx.indents = lx.indents[:len(lx.indents)-1]
		lx.emit(Dedent, lx.pos, lx.pos, lx.line, lx.line)
	}
	lx.emit(EOF, lx.pos, lx.pos, lx.line, lx.line)
}

// handleIndentation measures the indentation of a new logical line. It
// returns true when the line was blank or comment-only and has been consumed.
func (lx *lexer) handleIndentation() bool {
	column := 0
	index := lx.pos
	for index < len(lx.src) {
		switch lx.src[index] {
		case ' ':
			column++
		case '\t':
			column = (column/8 + 1) * 8
		case '\f':
			column = 0
		default:
			goto measured
		}
		index++
	}
measured:
	if index >= len(lx.src) {
		lx.pos = index
		return true
	}
	c := lx.src[index]
	if isNewline(c) {
		lx.pos = index
		lx.consumeNewline()
		return true
	}
	if c == '#' {
		lx.pos = index
		lx.comment()
		return true
	}
	if c == '\\' && index+1 < len(lx.src) && isNewline(lx.src[index+1]) {
		lx.pos = index + 1
		lx.consumeNewline()
		return true
	}
	lx.pos = index
	lx.atLineStart = false
	top := lx.indents[len(lx.indents)-1]
	switch {
	case column > top:
		lx.indents = append(lx.indents, column)
		lx.emit(Indent, index, index, lx.line, lx.line)
	case column < top:
		for column < lx.indents[len(lx.indents)-1] {
			lx.indents = lx.indents[:len(lx.indents)-1]
			lx.emit(Dedent, index, index, lx.line, lx.line)
		}
		if column != lx.indents[len(lx.indents)-1] {
			lx.fail("unindent does not match any outer indentation level")
		}
	}
	return false
}

func (lx *lexer) consumeNewline() {
	if lx.src[lx.pos] == '\r' && lx.pos+1 < len(lx.src) && lx.src[lx.pos+1] == '\n' {
		lx.pos += 2
	} else {
		lx.pos++
	}
	lx.line++
}

func (lx *lexer) newline() {
	start := lx.pos
	if lx.depth == 0 {
		if lx.lineHasTokens {
			lx.emit(Newline, start, start, lx.line, lx.line)
			lx.lineHasTokens = false
		}
		lx.atLineStart = true
	}
	lx.consumeNewline()
}

func (lx *lexer) comment() {
	start := lx.pos
	for lx.pos < len(lx.src) && !isNewline(lx.src[lx.pos]) {
		lx.pos++
	}
	lx.comments = append(lx.comments, Comment{Start: start, End: lx.pos})
}

func (lx *lexer) identifierOrString() {
	start := lx.pos
	for lx.pos < len(lx.src) && isIdentifierPart(lx.src, lx.pos) {
		_, size := utf8.DecodeRuneInString(lx.src[lx.pos:])
		lx.pos += size
	}
	if lx.pos < len(lx.src) && (lx.src[lx.pos] == '\'' || lx.src[lx.pos] == '"') {
		prefix := strings.ToLower(lx.src[start:lx.pos])
		if stringPrefixes[prefix] {
			lx.scanString(start, prefix)
			return
		}
	}
	lx.emit(Name, start, lx.pos, lx.line, lx.line)
}

func (lx *lexer) number() {
	start := lx.pos
	src := lx.src
	if src[lx.pos] == '0' && lx.pos+1 < len(src) && strings.ContainsRune("xXoObB", rune(src[lx.pos+1])) {
		lx.pos += 2
		for lx.pos < len(src) && (isHexDigit(src[lx.pos]) || src[lx.pos] == '_') {
			lx.pos++
		}
		lx.emit(Number, start, lx.pos, lx.line, lx.line)
		return
	}
	lx.scanDigits()
	if lx.pos < len(src) && src[lx.pos] == '.' {
		lx.pos++
		lx.scanDigits()
	}
	if lx.pos < len(src) && (src[lx.pos] == 'e' || src[lx.pos] == 'E') {
		next := lx.pos + 1
		if next < len(src) && (src[next] == '+' || src[next] == '-') {
			next++
		}
		if next < len(src) && isDigit(src[next]) {
			lx.pos = next
			lx.scanDigits()
		}
	}
	if lx.pos < len(src) && (src[lx.pos] == 'j' || src[lx.pos] == 'J') {
		lx.pos++
	}
	lx.emit(Number, start, lx.pos, lx.line, lx.line)
}

func (lx *lexer) scanDigits() {
	for lx.pos < len(lx.src) && (isDigit(lx.src[lx.pos]) || lx.src[lx.pos] == '_') {
		lx.pos++
	}
}

func (lx *lexer) operator() {
	for _, candidate := range operators {
		if strings.HasPrefix(lx.src[lx.pos:], candidate) {
			start := lx.pos
			lx.pos += len(candidate)
			switch candidate {
			case "(", "[", "{":
				lx.depth++
			case ")", "]", "}":
				if lx.depth == 0 {
					lx.fail("unmatched '%s'", candidate)
				}
				lx.depth--
			}
			lx.emit(Op, start, lx.pos, lx.line, lx.line)
			return
		}
	}
	r, _ := utf8.DecodeRuneInString(lx.src[lx.pos:])
	lx.fail("invalid character %q", r)
}

// scanString scans a string literal whose prefix (possibly empty) starts at
// start and whose opening quote is at the current position.
func (lx *lexer) scanString(start int, prefix string) {
	prefix = strings.ToLower(prefix)
	formatted := strings.ContainsAny(prefix, "ft")
	raw := strings.Contains(prefix, "r")
	quote := lx.src[lx.pos]
	triple := lx.pos+2 < len(lx.src) && lx.src[lx.pos+1] == quote && lx.src[lx.pos+2] == quote
	if triple {
		lx.pos += 3
	} else {
		lx.pos++
	}
	startLine := lx.line
	lx.scanStringBody(quote, triple, formatted, raw)
	lx.emit(String, start, lx.pos, startLine, lx.line)
}

// scanStringBody scans to the closing quote, skipping backslash pairs in every
// string kind. In f-strings a backslash before a brace is literal; \N{...} is not a field.
func (lx *lexer) scanStringBody(quote byte, triple, formatted, raw bool) {
	for {
		if lx.pos >= len(lx.src) {
			lx.fail("unterminated string literal")
		}
		c := lx.src[lx.pos]
		switch {
		case c == '\\':
			if lx.pos+1 >= len(lx.src) {
				lx.fail("unterminated string literal")
			}
			next := lx.src[lx.pos+1]
			if isNewline(next) {
				lx.pos++
				lx.consumeNewline()
				continue
			}
			if formatted && (next == '{' || next == '}') {
				lx.pos++
				continue
			}
			if formatted && !raw && next == 'N' && lx.pos+2 < len(lx.src) && lx.src[lx.pos+2] == '{' {
				if closing := strings.IndexByte(lx.src[lx.pos:], '}'); closing >= 0 {
					lx.pos += closing + 1
					continue
				}
			}
			lx.pos += 2
		case c == quote:
			if !triple {
				lx.pos++
				return
			}
			if lx.pos+2 < len(lx.src) && lx.src[lx.pos+1] == quote && lx.src[lx.pos+2] == quote {
				lx.pos += 3
				return
			}
			lx.pos++
		case isNewline(c):
			if !triple {
				lx.fail("unterminated string literal")
			}
			lx.consumeNewline()
		case formatted && c == '{':
			if lx.pos+1 < len(lx.src) && lx.src[lx.pos+1] == '{' {
				lx.pos += 2
				continue
			}
			lx.pos++
			lx.scanReplacementField(triple)
		case formatted && c == '}':
			if lx.pos+1 < len(lx.src) && lx.src[lx.pos+1] == '}' {
				lx.pos += 2
				continue
			}
			lx.pos++
		default:
			lx.pos++
		}
	}
}

// scanReplacementField consumes an f-string replacement field after its
// opening brace, including nested strings, brackets and format specs.
func (lx *lexer) scanReplacementField(triple bool) {
	depth := 0
	for {
		if lx.pos >= len(lx.src) {
			lx.fail("unterminated f-string replacement field")
		}
		c := lx.src[lx.pos]
		switch {
		case c == '\'' || c == '"':
			lx.scanNestedString()
		case c == '{' || c == '(' || c == '[':
			depth++
			lx.pos++
		case c == ')' || c == ']':
			depth--
			lx.pos++
		case c == '}':
			if depth == 0 {
				lx.pos++
				return
			}
			depth--
			lx.pos++
		case c == ':' && depth == 0:
			lx.pos++
			lx.scanFormatSpec(triple)
			return
		case c == '#':
			lx.comment()
		case isNewline(c):
			lx.consumeNewline()
		case c == '\\':
			lx.pos += 2
		default:
			lx.pos++
		}
	}
}

func (lx *lexer) scanFormatSpec(triple bool) {
	for {
		if lx.pos >= len(lx.src) {
			lx.fail("unterminated f-string format specifier")
		}
		c := lx.src[lx.pos]
		switch {
		case c == '{':
			lx.pos++
			lx.scanReplacementField(triple)
		case c == '}':
			lx.pos++
			return
		case isNewline(c):
			if !triple {
				lx.fail("unterminated f-string")
			}
			lx.consumeNewline()
		default:
			lx.pos++
		}
	}
}

// scanNestedString scans a string literal inside an f-string replacement
// field. Up to two preceding letters form its prefix.
func (lx *lexer) scanNestedString() {
	prefix := ""
	for index := lx.pos - 1; index >= 0 && index >= lx.pos-2 && isASCIILetter(lx.src[index]); index-- {
		prefix = string(lx.src[index]) + prefix
	}
	prefix = strings.ToLower(prefix)
	if !stringPrefixes[prefix] {
		prefix = ""
	}
	formatted := strings.ContainsAny(prefix, "ft")
	raw := strings.Contains(prefix, "r")
	quote := lx.src[lx.pos]
	triple := lx.pos+2 < len(lx.src) && lx.src[lx.pos+1] == quote && lx.src[lx.pos+2] == quote
	if triple {
		lx.pos += 3
	} else {
		lx.pos++
	}
	lx.scanStringBody(quote, triple, formatted, raw)
}

func isNewline(c byte) bool {
	return c == '\n' || c == '\r'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentifierStart(src string, pos int) bool {
	c := src[pos]
	if c < utf8.RuneSelf {
		return isASCIILetter(c) || c == '_'
	}
	r, _ := utf8.DecodeRuneInString(src[pos:])
	return unicode.IsLetter(r) || unicode.Is(unicode.Nl, r) || r == '_'
}

func isIdentifierPart(src string, pos int) bool {
	c := src[pos]
	if c < utf8.RuneSelf {
		return isASCIILetter(c) || isDigit(c) || c == '_'
	}
	r, _ := utf8.DecodeRuneInString(src[pos:])
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) ||
		unicode.Is(unicode.Nl, r) || unicode.Is(unicode.Pc, r) || r == '_'
}
