package syntax

import "strings"

// punctuators lists multi-character punctuators, longest first so the first
// match is the longest one. Anything else is a one-character punctuator.
var punctuators = []string{
	"...", "<<=", ">>=", "->*", "<=>",
	"::", "->", "++", "--", "<<", ">>", "<=", ">=", "==", "!=", "&&", "||",
	"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", ".*", "##",
}

var stringPrefixes = map[string]bool{"L": true, "u": true, "U": true, "u8": true}

var rawPrefixes = map[string]bool{"R": true, "LR": true, "uR": true, "UR": true, "u8R": true}

// tokenAt builds the token from start to the current position.
func (lx *lexer) tokenAt(kind Kind, start, line int) Token {
	return Token{Kind: kind, Text: lx.src[start:lx.pos], Start: start, End: lx.pos, Line: line, EndLine: lx.line}
}

// token lexes one token. Lenient lexing, used in directives, ends an unclosed
// quote at the end of the line instead of failing.
func (lx *lexer) token(lenient bool) Token {
	start, line := lx.pos, lx.line
	c := lx.src[lx.pos]
	switch {
	case isIdentifierByte(c) && !isDigit(c):
		return lx.tokenAt(lx.identifierOrLiteral(lenient), start, line)
	case isDigit(c) || (c == '.' && lx.pos+1 < len(lx.src) && isDigit(lx.src[lx.pos+1])):
		lx.number()
		return lx.tokenAt(Number, start, line)
	case c == '"' || c == '\'':
		lx.quoted(c, lenient)
		return lx.tokenAt(String, start, line)
	default:
		return lx.punct()
	}
}

// identifierOrLiteral reads an identifier, or a literal when the identifier
// is an encoding prefix directly followed by a quote.
func (lx *lexer) identifierOrLiteral(lenient bool) Kind {
	start := lx.pos
	for lx.pos < len(lx.src) && isIdentifierByte(lx.src[lx.pos]) {
		lx.pos++
	}
	if lx.pos >= len(lx.src) {
		return Identifier
	}
	word, next := lx.src[start:lx.pos], lx.src[lx.pos]
	switch {
	case next == '"' && rawPrefixes[word]:
		lx.rawString()
		return String
	case (next == '"' || next == '\'') && stringPrefixes[word]:
		lx.quoted(next, lenient)
		return String
	}
	return Identifier
}

// number reads a preprocessing number, which covers every numeric literal
// form including digit separators, exponents and suffixes.
func (lx *lexer) number() {
	lx.pos++
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		switch {
		case strings.IndexByte("eEpP", c) >= 0 && lx.pos+1 < len(lx.src) && (lx.src[lx.pos+1] == '+' || lx.src[lx.pos+1] == '-'):
			lx.pos += 2
		case c == '\'' && lx.pos+1 < len(lx.src) && isIdentifierByte(lx.src[lx.pos+1]):
			lx.pos += 2
		case isIdentifierByte(c) || c == '.':
			lx.pos++
		default:
			return
		}
	}
}

// quoted reads a string or character literal starting at its opening quote.
func (lx *lexer) quoted(quote byte, lenient bool) {
	line := lx.line
	lx.pos++
	for lx.pos < len(lx.src) {
		switch lx.src[lx.pos] {
		case '\\':
			lx.pos++
			if !lx.escapedNewline() {
				lx.pos++
			}
		case quote:
			lx.pos++
			return
		case '\n':
			if lenient {
				return
			}
			lx.fail(line, "unterminated %s literal", literalName(quote))
		default:
			lx.pos++
		}
	}
	lx.pos = min(lx.pos, len(lx.src))
	if !lenient {
		lx.fail(line, "unterminated %s literal", literalName(quote))
	}
}

// escapedNewline consumes the newline after a backslash inside a literal.
func (lx *lexer) escapedNewline() bool {
	switch {
	case lx.pos < len(lx.src) && lx.src[lx.pos] == '\n':
		lx.pos++
	case lx.pos+1 < len(lx.src) && lx.src[lx.pos] == '\r' && lx.src[lx.pos+1] == '\n':
		lx.pos += 2
	default:
		return false
	}
	lx.line++
	return true
}

func literalName(quote byte) string {
	if quote == '\'' {
		return "character"
	}
	return "string"
}

// rawString reads R"delimiter( ... )delimiter" starting at the quote.
func (lx *lexer) rawString() {
	line := lx.line
	open := strings.IndexByte(lx.src[lx.pos:], '(')
	if open < 0 || open > 17 || strings.ContainsAny(lx.src[lx.pos+1:lx.pos+open], " \\)\t\n") {
		lx.fail(line, "invalid raw string delimiter")
	}
	closing := ")" + lx.src[lx.pos+1:lx.pos+open] + "\""
	end := strings.Index(lx.src[lx.pos+open:], closing)
	if end < 0 {
		lx.fail(line, "unterminated raw string literal")
	}
	stop := lx.pos + open + end + len(closing)
	lx.line += strings.Count(lx.src[lx.pos:stop], "\n")
	lx.pos = stop
}

func (lx *lexer) punct() Token {
	start, line := lx.pos, lx.line
	for _, candidate := range punctuators {
		if strings.HasPrefix(lx.src[lx.pos:], candidate) {
			lx.pos += len(candidate)
			return lx.tokenAt(Punct, start, line)
		}
	}
	lx.pos++
	return lx.tokenAt(Punct, start, line)
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// isIdentifierByte accepts letters, digits, '_', '$' as compilers do, and
// every byte of a UTF-8 sequence, which covers extended identifiers.
func isIdentifierByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || isDigit(c) || c == '_' || c == '$' || c >= 0x80
}
