package cfamily

import (
	"sort"
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/cfamily/syntax"
)

// cloneTokens merges code tokens and directive tokens in source order, so a
// copied macro definition is a copy too. Comments are never tokens.
func cloneTokens(file *syntax.File) []adapter.Token {
	all := append([]syntax.Token(nil), file.Tokens...)
	for _, directive := range file.Directives {
		all = append(all, directive.Tokens...)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Start < all[j].Start })
	tokens := make([]adapter.Token, 0, len(all))
	for _, token := range all {
		tokens = append(tokens, adapter.Token{
			Type:    token.Kind.String(),
			Value:   token.Text,
			Line:    token.Line,
			EndLine: token.EndLine,
			Index:   len(tokens),
		})
	}
	return tokens
}

// collectImports gathers #include, #include_next and #import with a literal name, spelled
// with its delimiters as "a.h" or <a.h>; an include of a macro cannot be followed.
func collectImports(file *syntax.File) ([]adapter.Import, []adapter.LineSpan) {
	imports := []adapter.Import{}
	var spans []adapter.LineSpan
	for _, directive := range file.Directives {
		if directive.Name != "include" && directive.Name != "include_next" && directive.Name != "import" {
			continue
		}
		spans = append(spans, adapter.LineSpan{Line: directive.Line, EndLine: directive.EndLine})
		if len(directive.Tokens) < 3 || !isHeaderName(directive.Tokens[2]) {
			continue
		}
		imports = append(imports, adapter.Import{Specifier: directive.Tokens[2].Text, Line: directive.Line, Kind: adapter.ImportStatic})
	}
	return imports, spans
}

func isHeaderName(token syntax.Token) bool {
	text := token.Text
	return token.Kind == syntax.String && len(text) > 2 &&
		(strings.HasPrefix(text, "\"") && strings.HasSuffix(text, "\"") || strings.HasPrefix(text, "<") && strings.HasSuffix(text, ">"))
}
