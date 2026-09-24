package syntax

import (
	"strings"
	"testing"
)

func texts(tokens []Token) string {
	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		parts = append(parts, token.Text)
	}
	return strings.Join(parts, " ")
}

func mustTokenize(t *testing.T, src string) *File {
	t.Helper()
	file, err := Tokenize(src)
	if err != nil {
		t.Fatalf("Tokenize: %v", err)
	}
	return file
}

func TestTokensLiteralsAndPunctuators(t *testing.T) {
	file := mustTokenize(t, `int x = a<<=1 ? 'c' : u8"s\"q" L'\n' 1'000'000 0x1p+3 .5f;`+"\n"+
		`auto r = R"xy(a ")" b)xy"; p->*m; a <=> b; v...`)
	want := `int x = a <<= 1 ? 'c' : u8"s\"q" L'\n' 1'000'000 0x1p+3 .5f ; ` +
		`auto r = R"xy(a ")" b)xy" ; p ->* m ; a <=> b ; v ...`
	if got := texts(file.Tokens); got != want {
		t.Fatalf("tokens\n got %s\nwant %s", got, want)
	}
	if file.Tokens[4].Kind != Punct || file.Tokens[5].Kind != Number || file.Tokens[7].Kind != String {
		t.Errorf("kinds: %v %v %v", file.Tokens[4].Kind, file.Tokens[5].Kind, file.Tokens[7].Kind)
	}
}

func TestCommentsSplicesAndLines(t *testing.T) {
	src := "a /* one\ntwo */ b // tail \\\ncontinued\nc \\\n d\n\"x\\\ny\" e"
	file := mustTokenize(t, src)
	if got := texts(file.Tokens); got != "a b c d \"x\\\ny\" e" {
		t.Fatalf("tokens = %q", got)
	}
	if len(file.Comments) != 2 || src[file.Comments[1].Start:file.Comments[1].End] != "// tail \\\ncontinued" {
		t.Fatalf("comments = %+v", file.Comments)
	}
	lines := []int{1, 2, 4, 5, 6, 7}
	for i, token := range file.Tokens {
		if token.Line != lines[i] {
			t.Errorf("token %q on line %d, want %d", token.Text, token.Line, lines[i])
		}
	}
	if file.Tokens[4].EndLine != 7 {
		t.Errorf("spliced string ends on line %d", file.Tokens[4].EndLine)
	}
}

func TestDirectivesAreNotCodeTokens(t *testing.T) {
	file := mustTokenize(t, "#include <sys/stat.h>\n  # include \"a.h\" // why\n#define F(x) \\\n  ((x) + 1)\n#error don't\nint y;\n")
	if got := texts(file.Tokens); got != "int y ;" {
		t.Fatalf("tokens = %q", got)
	}
	if len(file.Directives) != 4 {
		t.Fatalf("directives = %d", len(file.Directives))
	}
	if got := texts(file.Directives[0].Tokens); got != "# include <sys/stat.h>" {
		t.Errorf("first directive = %q", got)
	}
	if file.Directives[1].Name != "include" || file.Directives[2].Name != "define" || file.Directives[2].EndLine != 4 {
		t.Errorf("directives = %+v", file.Directives)
	}
}

func TestFirstBranchOfEveryConditionalIsKept(t *testing.T) {
	src := `#ifdef A
one
#elif B
two
#else
three
#endif
#if 0
four ' unclosed
#elif C
five
#else
six
#endif
#if 0
seven
#endif
#ifndef D
# if 0
eight
# endif
nine
#else
ten
#endif
`
	file := mustTokenize(t, src)
	if got := texts(file.Tokens); got != "one five nine" {
		t.Fatalf("tokens = %q", got)
	}
}

func TestSkippedRegionsStillRecordComments(t *testing.T) {
	file := mustTokenize(t, "#if 0\n/* a */ x\n#endif\n")
	if len(file.Comments) != 1 || len(file.Tokens) != 0 {
		t.Fatalf("comments %+v tokens %+v", file.Comments, file.Tokens)
	}
}

func TestLexicalErrors(t *testing.T) {
	cases := map[string]string{
		"x\n/* open":              "unterminated comment (line 2)",
		"\"abc\n":                 "unterminated string literal (line 1)",
		"'a":                      "unterminated character literal (line 1)",
		"#if A\nx\n":              "#if has no matching #endif (line 1)",
		"#endif\n":                "#endif without #if (line 1)",
		"#if A\n#else\n#else\n":   "#else after #else (line 3)",
		"#if A\n#else\n#elif B\n": "#elif after #else (line 3)",
		"R\"x(abc\"":              "unterminated raw string literal (line 1)",
	}
	for src, want := range cases {
		if _, err := Tokenize(src); err == nil || err.Error() != want {
			t.Errorf("Tokenize(%q) error = %v, want %q", src, err, want)
		}
	}
}
