// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package parser

import (
	"slices"

	"github.com/lumiostack/lumioguard-cc/internal/thirdparty/tsgo/ast"
	"github.com/lumiostack/lumioguard-cc/internal/thirdparty/tsgo/core"
	"github.com/lumiostack/lumioguard-cc/internal/thirdparty/tsgo/scanner"
)

func getLanguageVariant(scriptKind core.ScriptKind) core.LanguageVariant {
	switch scriptKind {
	case core.ScriptKindTSX, core.ScriptKindJSX, core.ScriptKindJS, core.ScriptKindJSON:
		// .tsx and .jsx files are treated as jsx language variant.
		return core.LanguageVariantJSX
	}
	return core.LanguageVariantStandard
}

func tokenIsIdentifierOrKeyword(token ast.Kind) bool {
	return token >= ast.KindIdentifier
}

func tokenIsIdentifierOrKeywordOrGreaterThan(token ast.Kind) bool {
	return token == ast.KindGreaterThanToken || tokenIsIdentifierOrKeyword(token)
}

func GetJSDocCommentRanges(f *ast.NodeFactory, commentRanges []ast.CommentRange, node *ast.Node, text string) []ast.CommentRange {
	switch node.Kind {
	case ast.KindParameter, ast.KindTypeParameter, ast.KindFunctionExpression, ast.KindArrowFunction, ast.KindParenthesizedExpression, ast.KindVariableDeclaration, ast.KindExportSpecifier:
		for commentRange := range scanner.GetTrailingCommentRanges(f, text, node.Pos()) {
			commentRanges = append(commentRanges, commentRange)
		}
		for commentRange := range scanner.GetLeadingCommentRanges(f, text, node.Pos()) {
			commentRanges = append(commentRanges, commentRange)
		}
	default:
		for commentRange := range scanner.GetLeadingCommentRanges(f, text, node.Pos()) {
			commentRanges = append(commentRanges, commentRange)
		}
	}
	// Keep if the comment starts with '/**' but not if it is '/**/'
	return slices.DeleteFunc(commentRanges, func(comment ast.CommentRange) bool {
		commentStart := comment.Pos()
		commentLen := comment.End() - commentStart
		return comment.End() > node.End() || commentLen < 4 || text[commentStart+1] != '*' || text[commentStart+2] != '*' || text[commentStart+3] == '/'
	})
}

func isKeywordOrPunctuation(token ast.Kind) bool {
	return ast.IsKeywordKind(token) || ast.IsPunctuationKind(token)
}

func isJSDocLikeText(text string) bool {
	return len(text) >= 4 && text[1] == '*' && text[2] == '*' && text[3] != '/'
}
