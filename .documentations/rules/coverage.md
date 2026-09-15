---
description: Read line and branch coverage from the LCOV report your test tool already writes.
---

# Test coverage

Reports how much of your code your tests run, including only the lines you changed. lumioguard CC
**never runs your tests.** It reads the LCOV report your test tool already writes.

<div class="lg-summary" markdown>

| Rule | ID | Measures |
| --- | --- | --- |
| Line coverage | `coverage.line` | Share of executable lines the tests ran, per file |
| Branch coverage | `coverage.branch` | Share of branches the tests ran, per file |
| Changed-line coverage | `coverage.changed_line` | The same, for lines added since the comparison commit |
| Changed-branch coverage | `coverage.changed_branch` | The same, for branches on added lines |

All four are reported only. They never block.

</div>

## Turn it on

1. Make your test tool write an LCOV report:

    | Language | Tool |
    | --- | --- |
    | JavaScript and TypeScript | Jest, Vitest, c8 or nyc with the `lcov` reporter |
    | Python | `coverage lcov` from coverage.py |
    | Java | A JaCoCo report converted to LCOV |

2. Point the configuration at it:

    ```json
    "coverage": { "lcovFile": "coverage/lcov.info", "required": false }
    ```

3. Run your tests, then run `lumioguard-cc check --base main` to include changed-line coverage.

## How it works

- **Line and branch coverage** come from the report's `DA` and `BRDA` records, so your test tool
  decides what counts as executable. Branches marked `-` were never reached and are left out.
- **Changed-line and changed-branch coverage** count only lines added since the Git merge base, plus
  every line of untracked files. They need `--base`, because a stored baseline has no source code.
- **A file with nothing to cover** gets the status `not_applicable` and no value. It never shows as
  100 %.
- **Report paths** are matched relative to the report first, then to the project root.

## When the report cannot be used

| Diagnostic | Cause |
| --- | --- |
| `coverage.report_unavailable` | The report file is missing |
| `coverage.report_stale` | A source file changed after the report was written |
| `coverage.no_sources_mapped` | No path in the report matches an analyzed file |

With `"required": false` these are warnings. With `"required": true` they make the result
**incomplete** (exit code 2). Use `required` only in CI, straight after the tests run.

!!! note
    Coverage shows which code ran during the tests, not whether the tests checked the right things. The
    staleness check compares file times only.
