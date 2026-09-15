package java

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/java/syntax"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/sourcetext"
)

// tokenize converts the lexer output into clone-detection tokens (default
// channel only) and comment ranges (hidden channel comments).
func tokenize(p *parsedFile) ([]adapter.Token, []sourcetext.Range) {
	tokens := []adapter.Token{}
	var comments []sourcetext.Range
	for _, token := range p.tokens {
		switch {
		case token.GetTokenType() == antlr.TokenEOF:
			continue
		case token.GetChannel() == antlr.TokenDefaultChannel:
			text := token.GetText()
			tokens = append(tokens, adapter.Token{
				Type:    p.tokenName(token.GetTokenType()),
				Value:   text,
				Line:    token.GetLine(),
				EndLine: token.GetLine() + strings.Count(text, "\n"),
				Index:   len(tokens),
			})
		case token.GetTokenType() == syntax.JavaLexerCOMMENT || token.GetTokenType() == syntax.JavaLexerLINE_COMMENT:
			comments = append(comments, sourcetext.Range{Start: p.byteOffset(token.GetStart()), End: p.byteOffset(token.GetStop() + 1)})
		}
	}
	return tokens, comments
}
