# AGENTS.md

Instructions for coding agents working on lumioguard CC, a Go command-line tool that measures how
hard code is to change and whether a change made it worse. It supports JavaScript, TypeScript,
Python, Java and Go, runs offline, and never runs the code it analyzes. It checks its own code
through the root `.lumioguard-cc.json`: run `bin/lumioguard-cc check --base HEAD` before finishing.

Binary `lumioguard-cc`, config `.lumioguard-cc.json`, module `github.com/lumioguard/lumioguard-cc`.
Read [Development](.documentations/contributing/development.md) before changing code.

## Commands

```bash
go build -o bin/lumioguard-cc ./cmd/lumioguard-cc
go test ./...
go vet -unreachable=false ./...              # the generated Java parser trips only this check
gofmt -l .                                   # must print nothing
golangci-lint run ./...                      # must report 0 issues
sh .examples/run.sh bin/lumioguard-cc        # worked examples
make audit && make notices                   # after dependency changes
make docs                                    # after documentation changes; needs requirements-docs.txt
```

## Rules that must not regress

1. Incomplete is not clean: anything required not analyzed gives exit code 2, which beats 1.
2. Same code and configuration give the same report; only `run` varies.
3. No silent zeros: a missing value is `null` with a status and a reason.
4. Compare only like with like; only new or worsened blocking findings fail.
5. `check` never writes; only `init` and `baseline create` do.
6. `--format json` prints exactly one document on stdout; everything else goes to stderr.
7. Complexity is computed only in `internal/adapter/structure`; a counting change bumps the adapter `Version` and measurement `Variant`.
8. Adapters never guess: unresolvable internal imports are warnings, outside code is external.
9. Every dependency is its canonical upstream; copied code is recorded in `internal/thirdparty/components.go`.
10. Time alternatives in separate processes; the ANTLR parser caches state.

Reasons: [design principles](.documentations/contributing/development.md#design-principles).

## When you change something

- **A rule:** follow [adding or changing a rule](.documentations/contributing/development.md#adding-or-changing-a-rule).
- **A language:** follow [adding a language](.documentations/contributing/development.md#adding-a-language).
- **Report, config or baseline shape:** update `internal/domain`, `schemas/` and `schemas/schemas_test.go`; bump the schema version if not backward compatible.
- **A command:** a service in `internal/app`, a thin command in `internal/cli/commands.go`, tests in `internal/cli/cli_test.go`.
- **Anything users see:** update the matching `.documentations/` page and `CHANGELOG.md`.
- **Commands, flags, exit codes, the report or configuration:** update the guides in `internal/guide/topics/`. A test fails if a guide names a command or flag that does not exist, or shows defaults that differ from `init`.
- **Release file names or the install steps:** update `skills/lumioguard-cc/` too. The skill only installs the CLI and points to `lumioguard-cc guide`.
- **A design decision:** add it to [key decisions](.documentations/contributing/development.md#key-decisions).
- **Never edit by hand** `internal/thirdparty/tsgo/` or `internal/adapter/java/syntax/`; use their sync tools.
- **Never fix `.examples/*/before/`.** It is bad on purpose.

## Code

- Comments are at most two lines and say why. No commented-out code, no dead code.
- Exported identifiers have doc comments; wrap errors with context; sentinel errors live beside the code returning them.
- Problems in analyzed code are diagnostics, not Go errors.
- Tests use `testing`, `t.TempDir()` and real `git` with `t.Skip` when it is missing.
- Prefer the standard library.

## Documentation

- Plain English, short sentences, no marketing words.
- Numbers in examples come from running the built binary on that exact code.
- The README stays short; details go in `.documentations/`.
- Quote Mermaid labels that contain punctuation and use `<br/>` for line breaks.

## Contract with agents

| Exit code | Meaning |
| --- | --- |
| 0 | Required analysis completed and nothing blocking is new or worse |
| 1 | Analysis completed with new or worsened blocking findings |
| 2 | Incomplete analysis, invalid input or configuration, or internal failure |

`lumioguard-cc hook claude-stop` compares with `LUMIOGUARD_CC_BASELINE`, else `LUMIOGUARD_CC_BASE`,
else `HEAD`, and rejects both variables set at once. Its feedback lists only findings that fail and
names the exact command to reproduce them. Keep these properties, or the loop cannot end on a
repository that already has debt.
