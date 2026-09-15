// Package thirdparty records source copied or generated into this repository,
// which module scanners cannot see. The sync, audit and notices tools read it.
package thirdparty

// Component is one upstream project copied or generated into the repository.
type Component struct {
	// Module is the upstream Go module path, empty when the upstream is not a
	// Go module (an ANTLR grammar, for example).
	Module string
	// Name is the human-readable upstream project.
	Name string
	// Version is the pinned module version or source commit.
	Version string
	// License is the SPDX identifier of the upstream licence.
	License string
	// Path is the repository-relative directory holding the copy.
	Path string
	// Origin is the canonical upstream location.
	Origin string
	// Modifications states every change made to the upstream source, as
	// Apache-2.0 section 4(b) and the BSD licences require to be disclosed.
	Modifications []string
	// SyncTool regenerates the copy.
	SyncTool string
}

// TypeScriptParser is the TypeScript compiler front end used by the JS/TS adapter.
var TypeScriptParser = Component{
	Module:  "github.com/microsoft/typescript-go",
	Name:    "Microsoft TypeScript for Go (parser, scanner and AST packages)",
	Version: "v0.0.0-20260820064610-89d5d5b2849a",
	License: "Apache-2.0",
	Path:    "internal/thirdparty/tsgo",
	Origin:  "https://github.com/microsoft/typescript-go",
	Modifications: []string{
		"Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.",
		"The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.",
		"Test files omitted.",
	},
	SyncTool: "go run ./tools/tsgo-sync",
}

// JavaGrammar is the ANTLR grammar the Java parser is generated from.
var JavaGrammar = Component{
	Name:    "ANTLR grammars-v4 Java grammar (JavaLexer.g4, JavaParser.g4, java_parser_base.go)",
	Version: "942add78ce4402677ebd10ccd35ec07e28710b47",
	License: "BSD-3-Clause",
	Path:    "internal/adapter/java/syntax",
	Origin:  "https://github.com/antlr/grammars-v4/tree/master/java/java",
	Modifications: []string{
		"Parser and lexer generated from the grammar with the ANTLR tool; the generated Go code is committed.",
		"Semantic predicates rewritten from the Java target form (this.) to the Go form (p.), as the grammar's own transformGrammar.py does.",
		"java_parser_base.go copied with its package renamed to syntax.",
	},
	SyncTool: "go run ./tools/java-grammar-sync",
}

// ANTLRToolVersion is deliberately ahead of the 4.13.1 runtime: generated code
// never calls the runtime version check, and java-grammar-sync asserts that.
const ANTLRToolVersion = "4.13.2"

// All returns every recorded component.
func All() []Component {
	return []Component{TypeScriptParser, JavaGrammar}
}
