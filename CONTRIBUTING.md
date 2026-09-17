# Contributing to lumioguard CC

Contributions are welcome.

## Ways to help

- **Bug, wrong measurement or feature idea:** open an issue. For a wrong measurement, include the
  rule ID, the smallest code sample that shows it, and the value you expected.
- **Security problem:** follow [SECURITY.md](SECURITY.md). Never open a public issue for it.
- **Documentation fix:** open a pull request directly.

## Talk first

Open an issue before starting a new rule, a change to how a rule counts, a new language, a new
dependency, or a change to the report, configuration or baseline format. These change the numbers
people store, so they need agreement before code.

## Set up

You need Go 1.27 or later and Git.

```bash
git clone https://github.com/lumioguard/lumioguard-cc.git
cd lumioguard-cc
go build -o bin/lumioguard-cc ./cmd/lumioguard-cc
go test ./...
```

Read the [development guide](.documentations/contributing/development.md) before changing code. It covers the design
principles, code layout, adding a language or rule, testing and releasing.

## Checks before a pull request

The Build workflow runs these on Linux, macOS and Windows:

```bash
gofmt -l .                                 # must print nothing
go vet -unreachable=false ./...            # the generated Java parser trips only this check
go test ./...
golangci-lint run ./...                    # must report 0 issues
sh .examples/run.sh bin/lumioguard-cc      # worked examples still behave as documented
```

After a dependency change, also run `make audit` and `make notices`, and commit
`THIRD-PARTY-NOTICES.md`. After a documentation change, run `make docs` (see
[Testing](.documentations/contributing/development.md#testing)).

## Commits and pull requests

- One change per pull request, with tests.
- A change users can see updates the matching `.documentations/` page and `CHANGELOG.md`.
- First commit line in the imperative, under 72 characters, such as "Count labelled breaks in
  cognitive complexity". Explain why in the body when needed, and reference the issue.
- A maintainer reviews every pull request.
- Follow the code and documentation rules in [AGENTS.md](AGENTS.md). They apply to people too.

## AI coding agents

You may use them. You are responsible for every line: read it, run the checks, and make sure any
numbers in documentation came from running the tool. Agent instructions are in [AGENTS.md](AGENTS.md).
Claude Code reads `CLAUDE.md` files but not `AGENTS.md`: create a git-ignored `CLAUDE.local.md`
containing `@AGENTS.md` to load them.

## Licensing of contributions

The project is released under the [MIT License](LICENSE). By contributing, you agree that your
contribution is licensed under the same terms.
