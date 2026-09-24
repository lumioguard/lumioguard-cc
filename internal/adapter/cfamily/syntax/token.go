// Package syntax is a C and C++ lexer. It keeps the first branch of every
// preprocessor conditional, so the parser sees one balanced version of the file.
package syntax

import "fmt"

// Kind classifies a token. Keywords are identifiers; the parser knows them.
type Kind int

// Token kinds.
const (
	Identifier Kind = iota
	Number
	// String is a string or character literal, including prefixes and raw strings.
	String
	Punct
)

// String implements fmt.Stringer.
func (k Kind) String() string {
	switch k {
	case Identifier:
		return "Identifier"
	case Number:
		return "Number"
	case String:
		return "String"
	case Punct:
		return "Punct"
	default:
		return fmt.Sprintf("Kind(%d)", int(k))
	}
}

// Token is one lexical token with byte offsets and 1-based lines.
type Token struct {
	Kind    Kind
	Text    string
	Start   int
	End     int
	Line    int
	EndLine int
}

// Is reports whether the token is punctuation or an identifier spelled text.
func (t Token) Is(text string) bool {
	return (t.Kind == Punct || t.Kind == Identifier) && t.Text == text
}

// Range is a half-open byte range [Start, End).
type Range struct {
	Start int
	End   int
}

// Directive is one preprocessor directive in an analyzed region.
type Directive struct {
	// Name is the directive name, such as "include" or "define"; "" for a null directive.
	Name string
	// Tokens are the directive's tokens, starting with "#".
	Tokens  []Token
	Line    int
	EndLine int
}

// File is the lexed form of one source file.
type File struct {
	// Tokens are the code tokens of analyzed regions; directives are not among them.
	Tokens     []Token
	Directives []Directive
	// Comments are sorted by Start and include comments in skipped regions.
	Comments []Range
}

// Error is a lexical error at a line.
type Error struct {
	Line    int
	Message string
}

// Error implements error.
func (e *Error) Error() string {
	return fmt.Sprintf("%s (line %d)", e.Message, e.Line)
}
