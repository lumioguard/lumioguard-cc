# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

While the version is below 1.0, a minor release may change measurements or the
report format. Such changes are always listed under **Changed**, with what users
need to do.

## Unreleased

### Added

- `check`, `init`, `doctor`, `explain`, `guide` and `baseline create` commands. `guide` prints
  step-by-step instructions for setup, checking a change, cleanup, the report and configuration.
- Language support for JavaScript, TypeScript, Python 3, Java up to version 17 and Go,
  in one binary with no language runtime needed. Go imports are resolved through the
  project's `go.mod` files.
- Function checks: cyclomatic complexity, cognitive complexity, nesting depth,
  function length and parameter count.
- File and project size in source tokens (`size.file_tokens`, `size.total_tokens`), with an
  advisory `metrics.fileTokens` limit. The summary shows the total and, in a comparison, how
  many tokens the change added or removed.
- Repository checks: exact token duplication, module fan-in and fan-out,
  dependency cycles and declared architecture boundaries.
- `--format sarif` on `check`, for GitHub code scanning and other SARIF 2.1.0 readers.
- A GitHub Action, `lumiostack/lumioguard-cc@vX.Y.Z`, that installs the matching release and
  checks what a pull request made worse.
- Import of line, branch and changed-line coverage from LCOV reports.
- Comparison with a Git commit (`--base`) or a stored baseline (`--baseline`), so
  only new or worsened problems fail a check.
- Human summary and a versioned JSON report, with exit codes 0, 1 and 2.
- Claude Code Stop hook that compares with the last commit by default, and
  instructions for Codex and other agents.
- Worked examples in `.examples/` for each language.
- A documentation site with getting-started pages, task guides for coding agents, CI and clean-up,
  and a reference for every rule, command and setting.
- The `lumioguard-cc` agent skill, installable with `npx skills add`, which installs the CLI and
  points the agent to the CLI's guides.
- The MIT license.
