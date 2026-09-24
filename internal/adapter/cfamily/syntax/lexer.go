package syntax

import "fmt"

type lexError struct {
	err *Error
}

type lexer struct {
	src  string
	pos  int
	line int
	// lineHasToken is false until the current line has something other than
	// whitespace and comments, so "#" there starts a directive.
	lineHasToken bool
	conditionals []conditional
	out          File
}

// Tokenize splits C or C++ source into tokens, active directives and comment
// ranges. Code in skipped conditional branches produces no tokens.
func Tokenize(src string) (file *File, err error) {
	lx := &lexer{src: src, line: 1}
	defer func() {
		if recovered := recover(); recovered != nil {
			failure, ok := recovered.(lexError)
			if !ok {
				panic(recovered)
			}
			file, err = nil, failure.err
		}
	}()
	lx.run()
	return &lx.out, nil
}

func (lx *lexer) fail(line int, format string, args ...any) {
	panic(lexError{err: &Error{Line: line, Message: fmt.Sprintf(format, args...)}})
}

func (lx *lexer) run() {
	if len(lx.src) >= 3 && lx.src[:3] == "\xEF\xBB\xBF" {
		lx.pos = 3
	}
	for lx.pos < len(lx.src) {
		if lx.layout() {
			continue
		}
		switch {
		case lx.src[lx.pos] == '#' && !lx.lineHasToken:
			lx.directive()
		case lx.active():
			token := lx.token(false)
			lx.out.Tokens = append(lx.out.Tokens, token)
			lx.lineHasToken = true
		default:
			lx.skipCharacter()
		}
	}
	if len(lx.conditionals) > 0 {
		open := lx.conditionals[len(lx.conditionals)-1]
		lx.fail(open.line, "#%s has no matching #endif", open.name)
	}
}

// layout consumes one piece of whitespace, a line splice or a comment.
func (lx *lexer) layout() bool {
	c := lx.src[lx.pos]
	switch {
	case c == '\n':
		lx.pos++
		lx.line++
		lx.lineHasToken = false
	case c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v':
		lx.pos++
	case lx.splice():
	case lx.comment():
	default:
		return false
	}
	return true
}

// splice consumes a backslash-newline, which joins two physical lines.
func (lx *lexer) splice() bool {
	if lx.src[lx.pos] != '\\' {
		return false
	}
	switch {
	case lx.pos+1 < len(lx.src) && lx.src[lx.pos+1] == '\n':
		lx.pos += 2
	case lx.pos+2 < len(lx.src) && lx.src[lx.pos+1] == '\r' && lx.src[lx.pos+2] == '\n':
		lx.pos += 3
	default:
		return false
	}
	lx.line++
	return true
}

func (lx *lexer) comment() bool {
	if lx.pos+1 >= len(lx.src) || lx.src[lx.pos] != '/' {
		return false
	}
	start := lx.pos
	switch lx.src[lx.pos+1] {
	case '*':
		lx.blockComment()
	case '/':
		lx.lineComment()
	default:
		return false
	}
	lx.out.Comments = append(lx.out.Comments, Range{Start: start, End: lx.pos})
	return true
}

func (lx *lexer) blockComment() {
	line := lx.line
	lx.pos += 2
	for lx.pos+1 < len(lx.src) {
		if lx.src[lx.pos] == '*' && lx.src[lx.pos+1] == '/' {
			lx.pos += 2
			return
		}
		if lx.src[lx.pos] == '\n' {
			lx.line++
		}
		lx.pos++
	}
	lx.fail(line, "unterminated comment")
}

// lineComment stops before the newline; a splice continues the comment.
func (lx *lexer) lineComment() {
	for lx.pos < len(lx.src) && lx.src[lx.pos] != '\n' {
		if !lx.splice() {
			lx.pos++
		}
	}
}

// skipCharacter passes over text in a skipped branch. Quotes there need not
// be closed, as compilers accept, so a literal ends at the line's end.
func (lx *lexer) skipCharacter() {
	c := lx.src[lx.pos]
	lx.lineHasToken = true
	if c == '"' || c == '\'' {
		lx.quoted(c, true)
		return
	}
	lx.pos++
}

// directive lexes one directive line and applies it to the conditional stack.
func (lx *lexer) directive() {
	directive := Directive{Line: lx.line}
	lx.pos++
	directive.Tokens = append(directive.Tokens, lx.tokenAt(Punct, lx.pos-1, lx.line))
	for lx.pos < len(lx.src) && lx.src[lx.pos] != '\n' {
		if lx.layout() {
			continue
		}
		if lx.headerNameFollows(directive.Tokens) {
			directive.Tokens = append(directive.Tokens, lx.headerName())
			continue
		}
		directive.Tokens = append(directive.Tokens, lx.token(true))
	}
	directive.EndLine = lx.line
	if len(directive.Tokens) > 1 && directive.Tokens[1].Kind == Identifier {
		directive.Name = directive.Tokens[1].Text
	}
	if !lx.conditional(&directive) && lx.active() {
		lx.out.Directives = append(lx.out.Directives, directive)
	}
	lx.lineHasToken = true
}

func (lx *lexer) headerNameFollows(tokens []Token) bool {
	if len(tokens) != 2 || lx.src[lx.pos] != '<' {
		return false
	}
	name := tokens[1].Text
	return name == "include" || name == "include_next" || name == "import"
}

// headerName lexes <path> as one String token, as the standard's header-name.
func (lx *lexer) headerName() Token {
	start := lx.pos
	for lx.pos < len(lx.src) && lx.src[lx.pos] != '\n' {
		lx.pos++
		if lx.src[lx.pos-1] == '>' {
			break
		}
	}
	return lx.tokenAt(String, start, lx.line)
}
