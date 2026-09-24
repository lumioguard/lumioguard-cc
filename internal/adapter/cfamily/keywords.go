package cfamily

func set(words ...string) map[string]bool {
	out := make(map[string]bool, len(words))
	for _, word := range words {
		out[word] = true
	}
	return out
}

// typeKeywords are the built-in type words and qualifiers; "&&" after one of
// them declares a reference instead of joining conditions.
var typeKeywords = set(
	"auto", "void", "char", "short", "int", "long", "float", "double", "signed", "unsigned",
	"bool", "_Bool", "wchar_t", "char8_t", "char16_t", "char32_t", "const", "volatile",
)

// attributeWords take a parenthesised argument that is never a parameter list.
var attributeWords = set(
	"__attribute__", "__attribute", "__declspec", "alignas", "_Alignas", "decltype", "noexcept",
	"throw", "sizeof", "alignof", "_Alignof", "typeof", "__typeof__", "__typeof", "typeof_unqual",
	"requires", "asm", "__asm__", "__asm", "static_assert", "_Static_assert", "explicit",
	"_Pragma", "__pragma", "_Generic", "__builtin_offsetof", "offsetof", "_Atomic",
)

// reserved words never name a function.
var reserved = merge(typeKeywords, attributeWords, set(
	"if", "else", "for", "while", "do", "switch", "case", "default", "return", "break", "continue",
	"goto", "static", "extern", "inline", "register", "typedef", "struct", "union", "enum",
	"class", "template", "typename", "using", "namespace", "operator", "new", "delete", "this",
	"true", "false", "nullptr", "try", "catch", "virtual", "friend", "constexpr", "consteval",
	"constinit", "mutable", "thread_local", "_Thread_local", "restrict", "__restrict", "__restrict__",
	"co_await", "co_return", "co_yield", "public", "private", "protected", "export", "concept",
	"final", "override", "__inline", "__inline__", "__forceinline", "_Noreturn", "__extension__",
))

// cppOnly are C++ keywords that C code uses as names, such as zlib's try(). Words like
// decltype stay reserved, because .h headers are read as C but often hold C++.
var cppOnly = set("class", "template", "typename", "using", "namespace", "new", "delete", "this",
	"try", "catch", "virtual", "friend", "public", "private", "protected", "export", "concept",
	"final", "override", "mutable")

// qualifierWords may follow a parameter list before the body.
var qualifierWords = set("const", "volatile", "&", "&&", "override", "final", "mutable",
	"constexpr", "consteval", "static", "__restrict", "__restrict__", "restrict")

// statementWords start a statement, so they end a preceding macro invocation
// that has no semicolon.
var statementWords = set("if", "for", "while", "do", "switch", "return", "break", "continue",
	"goto", "case", "else", "co_return")

// operandEnders are keywords that cannot end an operand.
var operandEnders = set("return", "case", "new", "delete", "throw", "sizeof", "alignof",
	"co_await", "co_yield", "co_return", "else", "do", "typeof", "decltype", "noexcept")

var assignmentOperators = set("=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=")

var classKeys = set("class", "struct", "union")

// accessSpecifiers may be followed by Qt's "slots" before the colon.
var accessSpecifiers = set("public", "protected", "private")

// accessWords start an access label in a class body, including Qt's labels.
var accessWords = merge(accessSpecifiers, set("signals", "Q_SIGNALS", "slots", "Q_SLOTS"))

func merge(sets ...map[string]bool) map[string]bool {
	out := map[string]bool{}
	for _, words := range sets {
		for word := range words {
			out[word] = true
		}
	}
	return out
}
