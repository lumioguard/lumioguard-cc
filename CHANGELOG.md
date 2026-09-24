# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

While the version is below 1.0, a minor release may change measurements or the
report format. Such changes are always listed under **Changed**, with what users
need to do.

## 0.2.0 - 2026-09-24

### Added

- C and C++ support: `.c` and `.h` count as C; `.cc`, `.cpp`, `.cxx`, `.c++`, `.hh`, `.hpp`,
  `.hxx` and `.h++` count as C++. A parser written for this tool reads the code without running the
  preprocessor: of each `#if` group only the first branch is read, or the next one after `#if 0`.
  Functions, methods, constructors, operators and lambdas are measured with the same rules as the
  other languages. `#include` directives become dependencies, resolved without include paths; an
  include that matches two files is reported as unresolved. K&R-style definitions and Objective-C
  are not supported and fail with `cfamily.parse_failed`.
- Worked examples for C (`.examples/c-sensor-pipeline`) and C++ (`.examples/cpp-shipping-quotes`),
  each 9 blocking findings before and 0 after; `.examples/run.sh` runs them with the others.

### Changed

- The default `source.include` patterns now cover the C and C++ extensions, which changes the
  default configuration hash.
- The report lists the new `cfamily` adapter. Because a stored baseline records every adapter, a
  baseline created with 0.1.0 now fails with `baseline.analyzers_incompatible`, even in a project
  without C or C++. After upgrading, review and recreate stored baselines with
  `lumioguard-cc baseline create`. Comparisons with `--base` are not affected. In a project that
  should not analyze C or C++, set `source.include` to its own languages.

## 0.1.0 - 2026-09-17

### Added

- `check`, `init`, `doctor`, `explain`, `guide` and `baseline create` commands. `guide` prints
  step-by-step instructions for setup, checking a change, cleanup, the report and configuration.
- `worklist`, which orders the places to fix for a cleanup: cycles and boundary violations, then
  functions and files by how many rules they break, with nested functions under their function,
  then duplicated blocks by size. `--rule`, `--path` and `--top` narrow it; `--format json` gives
  the same list to tools.
- Language support for JavaScript, TypeScript, Python 3, Java up to version 17 and Go,
  in one binary with no language runtime needed. Go imports are resolved through the
  project's `go.mod` files.
- Function checks: cyclomatic complexity, cognitive complexity, nesting depth,
  function length and parameter count.
- File and project size in source tokens (`size.file_tokens`, `size.total_tokens`), with an
  advisory `metrics.fileTokens` limit. The summary shows the total and, in a comparison, how
  many tokens the change added or removed.
- Repository checks: exact token duplication, module fan-in and fan-out,
  dependency cycles and declared architecture boundaries. A copied block is one
  finding however long it is, identified by the files and functions that hold
  the copies, with the lines the copies take up as its value. Import
  declarations never count as copies.
- Default source exclusions for dependency, build, cache and virtual-environment
  directories, including the output of Next.js, Nuxt, SvelteKit, Turborepo and
  Parcel.
- `--format sarif` on `check`, for GitHub code scanning and other SARIF 2.1.0 readers.
- A GitHub Action, `lumioguard/lumioguard-cc@vX.Y.Z`, that installs the matching release and
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
