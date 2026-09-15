---
description: Every check lumioguard CC runs, its default, and how a finding is decided.
---

# Rules

Every check lumioguard CC runs, with its ID, its default threshold and whether it blocks by default.
Run `lumioguard-cc explain <ID>` for the exact definition of any of them.

## All checks

| Check | ID | What it looks at | Default | Blocks by default |
| --- | --- | --- | --- | --- |
| [Cyclomatic complexity](complexity.md#cyclomatic-complexity) | `complexity.cyclomatic` | Decision points in a function | 12 | No |
| [Cognitive complexity](complexity.md#cognitive-complexity) | `complexity.cognitive` | How hard a function is to follow | 15 | No |
| [Nesting depth](complexity.md#nesting-depth) | `complexity.nesting_depth` | Deepest nesting in a function | 4 | No |
| [Function length](size.md#function-length) | `size.function_lines` | Lines of code in a function | 80 | No |
| [Parameter count](size.md#parameter-count) | `size.parameter_count` | Parameters a function takes | 5 | No |
| [File tokens](size.md#file-tokens) | `size.file_tokens` | Source tokens in a file | 4000 | No |
| [Total tokens](size.md#file-tokens) | `size.total_tokens` | Source tokens in the whole project | Reported only | Never |
| [Duplication](duplication.md) | `duplication.token_clone_density` | Share of lines inside exact copies | 5 % | No |
| [Fan-out](dependencies.md#fan-out-and-fan-in) | `coupling.module_fan_out` | Project files one file imports | 20 | No |
| [Fan-in](dependencies.md#fan-out-and-fan-in) | `coupling.module_fan_in` | Project files that import one file | Reported only | Never |
| [Dependency cycles](dependencies.md#dependency-cycles) | `dependency.cycle` | Files that import each other in a loop | Any cycle | Yes |
| [Boundary violations](boundaries.md) | `architecture.boundary_violation` | Imports your declared layers forbid | Any violation | Yes |
| [Test coverage](coverage.md) | `coverage.line`, `coverage.branch`, `coverage.changed_line`, `coverage.changed_branch` | Coverage from an LCOV report | Reported only | Never |

Size and complexity rules only warn by default, because there is no universal right limit. Cycles block
because they are a structural fault. Boundary rules block because they exist only once you declare
them.

## How a finding is decided

- **The value must be strictly above the threshold.** A threshold of 12 allows 12 and reports 13.
- **The severity is only a label.** `info`, `warning` and `error` decide nothing. `block` decides
  whether a finding can fail the check.
- **Only new or worse findings fail.** When you compare with a Git commit or a stored baseline, a
  blocking finding fails the check only if it is new or its value went up. Without a comparison, every
  finding counts as new.
- **A disabled rule** (`"enabled": false`) still records its measurement, but raises no finding.

## Which functions are measured

Complexity and size rules measure every function with a body: functions, methods, constructors, arrow
functions and lambdas. A nested function is also measured on its own.

Results name a function by its class and enclosing function, such as `OrderService.submit` or
`Invoice.<init>`. Unnamed functions appear as `<anonymous>` or `<lambda>`, and a repeated name gets a
suffix such as `#2`.

## Rule groups

<div class="grid cards" markdown>

-   [:lucide-git-branch:{ .lg-icon } **Complexity**](complexity.md)

    Cyclomatic complexity, cognitive complexity and nesting depth.

-   [:lucide-file-code:{ .lg-icon } **Size**](size.md)

    Function length, parameter count and file size in tokens.

-   [:lucide-copy:{ .lg-icon } **Duplication**](duplication.md)

    Blocks of code copied in more than one place.

-   [:lucide-git-compare:{ .lg-icon } **Dependencies**](dependencies.md)

    Fan-out, fan-in and dependency cycles.

-   [:lucide-layers:{ .lg-icon } **Architecture boundaries**](boundaries.md)

    Imports between layers that your team has ruled out.

-   [:lucide-shield-check:{ .lg-icon } **Test coverage**](coverage.md)

    Line and branch coverage from your test tool's report.

</div>

A passing check does not mean the code is correct, secure or well tested. See
[what a check does not prove](../reference/languages.md#what-a-check-does-not-prove).
