package golang

import (
	"go/scanner"
	"go/token"
	"strings"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/sourcetext"
)

// tokenize produces the clone-detection token stream and every comment range.
// Comments are never tokens. An automatic semicolon becomes ";" so a block
// matches whether or not its statements end in explicit semicolons.
func tokenize(code string) ([]adapter.Token, []sourcetext.Range) {
	fset := token.NewFileSet()
	file := fset.AddFile("", -1, len(code))
	var s scanner.Scanner
	s.Init(file, []byte(code), func(token.Position, string) {}, scanner.ScanComments)

	tokens := []adapter.Token{}
	var comments []sourcetext.Range
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		offset := file.Offset(pos)
		if tok == token.COMMENT {
			comments = append(comments, sourcetext.Range{Start: offset, End: offset + len(lit)})
			continue
		}
		value := lit
		switch {
		case tok == token.SEMICOLON:
			value = ";"
		case value == "":
			value = tok.String()
		}
		line := file.Line(pos)
		tokens = append(tokens, adapter.Token{
			Type:    tok.String(),
			Value:   value,
			Line:    line,
			EndLine: line + strings.Count(lit, "\n"),
			Index:   len(tokens),
		})
	}
	return tokens, comments
}
