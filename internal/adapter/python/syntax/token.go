// Package syntax is a Python 3 lexer and parser producing tokens, comment ranges
// and a positioned CPython-like AST. Python 2 syntax is not supported.
package syntax

import "fmt"

// TokenType classifies a token.
type TokenType int

// Token types. Newline, Indent and Dedent are structural tokens with empty text.
const (
	EOF TokenType = iota
	Newline
	Indent
	Dedent
	Name
	Number
	String
	Op
)

// String implements fmt.Stringer.
func (t TokenType) String() string {
	switch t {
	case EOF:
		return "EOF"
	case Newline:
		return "Newline"
	case Indent:
		return "Indent"
	case Dedent:
		return "Dedent"
	case Name:
		return "Name"
	case Number:
		return "Number"
	case String:
		return "String"
	case Op:
		return "Op"
	default:
		return fmt.Sprintf("TokenType(%d)", int(t))
	}
}

// Token is one lexical token with byte offsets and 1-based lines.
type Token struct {
	Type    TokenType
	Text    string
	Start   int
	End     int
	Line    int
	EndLine int
}

// Comment is the byte range of one comment, including the leading '#'.
type Comment struct {
	Start int
	End   int
}

// Error is a lexical or syntax error at a line.
type Error struct {
	Line    int
	Message string
}

// Error implements error.
func (e *Error) Error() string {
	return fmt.Sprintf("%s (line %d)", e.Message, e.Line)
}
