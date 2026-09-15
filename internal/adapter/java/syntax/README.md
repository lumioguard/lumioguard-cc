# Generated Java parser

Generated code. Do not edit by hand; regenerate with:

    go run ./tools/java-grammar-sync

Source grammar: antlr/grammars-v4, path `java/java`, commit `942add78ce4402677ebd10ccd35ec07e28710b47`.
Licence: BSD-3-Clause, reproduced in full in LICENSE beside this file.
Generator: ANTLR 4.13.2. Runtime: `github.com/antlr4-go/antlr/v4 v4.13.1`, the latest
published Go runtime. The tool is deliberately ahead; the sync tool asserts that
the generated code never calls the runtime's version check, which is the only
way the difference could become observable.

## Modifications

- Parser and lexer generated from the grammar with the ANTLR tool; the generated Go code is committed.
- Semantic predicates rewritten from the Java target form (this.) to the Go form (p.), as the grammar's own transformGrammar.py does.
- java_parser_base.go copied with its package renamed to syntax.

The pinned commit is recorded in `internal/thirdparty/components.go`.
