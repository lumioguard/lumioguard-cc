package typescript

import (
	"sort"
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/sourcetext"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/ast"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/core"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/diagnostics"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty/tsgo/scanner"
)

// lexemes is the token stream used for clone detection plus every comment
// range, which function size needs.
type lexemes struct {
	tokens   []adapter.Token
	comments []sourcetext.Range
}

type opaqueRange struct {
	pos  int
	end  int
	kind ast.Kind
}

// opaqueKinds lex differently by parser context (regular expressions, template
// parts, JSX text), so each becomes one token with its range taken from the AST.
var opaqueKinds = map[ast.Kind]bool{
	ast.KindRegularExpressionLiteral:      true,
	ast.KindNoSubstitutionTemplateLiteral: true,
	ast.KindTemplateHead:                  true,
	ast.KindTemplateMiddle:                true,
	ast.KindTemplateTail:                  true,
	ast.KindJsxText:                       true,
}

// tokenize produces the token stream with the TypeScript scanner, guided by
// the AST for context-dependent constructs. Comments are never tokens.
func tokenize(p *parsedFile) lexemes {
	result := lexemes{tokens: []adapter.Token{}}
	opaque := collectOpaqueRanges(p)
	text := p.text

	s := scanner.NewScanner()
	s.SetText(text)
	if p.jsx {
		s.SetLanguageVariant(core.LanguageVariantJSX)
	}
	s.SetOnError(func(*diagnostics.Message, int, int, ...any) {})

	cursor, next := 0, 0
	for {
		limit := len(text)
		if next < len(opaque) {
			limit = opaque[next].pos
		}
		start := scanner.SkipTrivia(text, cursor)
		if start > limit {
			start = limit
		}
		result.comments = append(result.comments, scanComments(text, cursor, start)...)

		if next < len(opaque) && start >= opaque[next].pos {
			current := opaque[next]
			next++
			if current.end <= cursor {
				continue
			}
			value := text[current.pos:current.end]
			if current.kind == ast.KindJsxText {
				value = strings.TrimSpace(value)
			}
			if value != "" {
				result.tokens = append(result.tokens, p.token(kindName(current.kind), value, current.pos, current.end, len(result.tokens)))
			}
			cursor = current.end
			continue
		}
		if start >= len(text) {
			break
		}
		s.ResetTokenState(start)
		kind := s.Scan()
		if kind == ast.KindEndOfFile || s.TokenEnd() <= start {
			break
		}
		end := s.TokenEnd()
		if next < len(opaque) && end > opaque[next].pos {
			end = opaque[next].pos
		}
		result.tokens = append(result.tokens, p.token(kindName(kind), text[s.TokenStart():end], s.TokenStart(), end, len(result.tokens)))
		cursor = end
	}
	return result
}

func (p *parsedFile) token(kind, value string, pos, end, index int) adapter.Token {
	return adapter.Token{Type: kind, Value: value, Line: p.line(pos), EndLine: p.line(max(end-1, pos)), Index: index}
}

func collectOpaqueRanges(p *parsedFile) []opaqueRange {
	var ranges []opaqueRange
	var walk func(node *ast.Node)
	visit := func(node *ast.Node) bool {
		walk(node)
		return false // ForEachChild stops on true; every child is needed
	}
	walk = func(node *ast.Node) {
		if opaqueKinds[node.Kind] {
			start := node.Pos()
			if node.Kind != ast.KindJsxText {
				start = p.tokenStart(node)
			}
			if node.End() > start {
				ranges = append(ranges, opaqueRange{pos: start, end: node.End(), kind: node.Kind})
			}
			return
		}
		node.ForEachChild(visit)
	}
	walk(p.source.AsNode())
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].pos < ranges[j].pos })
	return ranges
}

// scanComments finds comments inside a trivia-only region of the text.
func scanComments(text string, from, to int) []sourcetext.Range {
	var comments []sourcetext.Range
	for index := from; index < to; {
		if text[index] == '/' && index+1 < to && text[index+1] == '/' {
			end := index + 2
			for end < to && text[end] != '\n' && text[end] != '\r' {
				end++
			}
			comments = append(comments, sourcetext.Range{Start: index, End: end})
			index = end
			continue
		}
		if text[index] == '/' && index+1 < to && text[index+1] == '*' {
			end := to
			if closing := strings.Index(text[index+2:to], "*/"); closing >= 0 {
				end = index + 2 + closing + 2
			}
			comments = append(comments, sourcetext.Range{Start: index, End: end})
			index = end
			continue
		}
		index++
	}
	return comments
}
