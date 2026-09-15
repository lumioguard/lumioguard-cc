---
description: How to report problems, suggest changes and contribute code to lumioguard CC.
---

# Contribute

lumioguard CC is open to contributions: bug reports, wrong measurements, documentation fixes and code.

## Ways to help

<div class="grid cards" markdown>

-   :lucide-circle-help:{ .lg .middle .lg-icon } **Report a bug or a wrong number**

    ---

    [Open an issue](https://github.com/lumiostack/lumioguard-cc/issues). For a wrong measurement,
    include the rule ID, the smallest code sample that shows it, and the value you expected.

-   :lucide-book-open:{ .lg .middle .lg-icon } **Improve the documentation**

    ---

    Every page has an edit button at the top. Small fixes can go straight to a pull request.

-   :lucide-code-xml:{ .lg .middle .lg-icon } **Contribute code**

    ---

    Read the development guide first. It covers the design, the code layout, tests and releases.

    [:octicons-arrow-right-24: Development guide](development.md)

-   :lucide-lock:{ .lg .middle .lg-icon } **Report a security issue**

    ---

    Follow the [security policy](https://github.com/lumiostack/lumioguard-cc/blob/main/SECURITY.md).
    Never open a public issue for it.

</div>

## Talk first about big changes

Open an issue before you start on:

- a new rule, or a change to how a rule counts;
- a new language;
- a new dependency;
- a change to the report, configuration or baseline format.

These change the numbers people store and the files they depend on, so they need agreement before
code.

## Your first pull request

You need Go 1.27 or later and Git.

```bash
git clone https://github.com/lumiostack/lumioguard-cc.git
cd lumioguard-cc
go build -o bin/lumioguard-cc ./cmd/lumioguard-cc
go test ./...
```

Before you open the pull request:

- [x] One change per pull request, with tests.
- [x] `gofmt -l .` prints nothing, and `golangci-lint run ./...` reports 0 issues.
- [x] A change users can see updates the matching documentation page and `CHANGELOG.md`.
- [x] Any number in the documentation comes from running the tool.

The full checklist is in
[CONTRIBUTING.md](https://github.com/lumiostack/lumioguard-cc/blob/main/CONTRIBUTING.md).

## Using AI coding agents

You may use them. You are responsible for every line: read it, run the checks, and make sure any
numbers in the documentation came from running the tool. Agent instructions are in
[AGENTS.md](https://github.com/lumiostack/lumioguard-cc/blob/main/AGENTS.md).
