# The JSON report

`lumioguard-cc check --format json` prints one JSON document on stdout. Errors go to stderr. The
same code and configuration always give the same report, except for the `run` block.

## Top level

| Field | Contents |
| --- | --- |
| `policy.status` | `passed`, `failed` or `incomplete` (exit codes 0, 1 and 2) |
| `policy.blockingFindings` | Blocking findings that are new or worse |
| `policy.warningFindings` | Non-blocking findings that are still present |
| `comparison` | `mode` (`none`, `git` or `baseline`), `reference` and `comparable` |
| `findings` | Every finding, including `resolved` ones |
| `measurements` | Every value measured, including values under their threshold |
| `diagnostics` | Problems with the analysis itself. `"required": true` makes the result incomplete. |
| `scope` | `filesDiscovered`, `filesAnalyzed` and a count per language |

The measurement with `metricId` `size.total_tokens` is the project's size in source tokens. With a
comparison, its `delta` is how much code the change added or removed; keep it small.

## A finding

| Field | Meaning |
| --- | --- |
| `ruleId` | The rule, such as `complexity.cognitive` |
| `blocking` | Whether this rule can fail the check |
| `classification` | `new`, `worsened`, `existing` or `resolved` |
| `current`, `threshold` | The measured value and the configured limit |
| `baseline`, `delta` | The earlier value and the change, when the check compared with something |
| `scope` | Where the finding is: `kind`, plus `file`, `symbol`, `line` and `endLine` for functions |
| `evidence` | Why the number is what it is |
| `message` | A one-line description |

## What did my change do?

After a check with `--base` or `--baseline`, sort the findings:

| Group | Filter | Action |
| --- | --- | --- |
| Fails the check | `blocking` is true and `classification` is `new` or `worsened` | Fix before finishing |
| Made worse but advisory | `blocking` is false and `classification` is `new` or `worsened` | Fix if reasonable, otherwise tell the user |
| Already there | `classification` is `existing` | Leave alone, unless the task is a cleanup |
| Fixed | `classification` is `resolved` | Mention it in a cleanup report |

Deal with required diagnostics before any of these. Until they are fixed, part of the code was not
analyzed.

## Where a finding points

| `scope.kind` | Rules | Location |
| --- | --- | --- |
| `function` | Complexity and size rules | `file`, `symbol`, `line`, `endLine` |
| `file` | `coupling.module_fan_out`, `size.file_tokens` | `file` |
| `dependency` | `architecture.boundary_violation` | `evidence.source`, `evidence.target`, `evidence.line` |
| `cycle` | `dependency.cycle` | `evidence.modules` |
| `repository` | `duplication.token_clone`, `duplication.token_clone_density` | `evidence.occurrences`: `file`, `startLine`, `endLine` and, inside a function, `symbol` for each copy |

## Evidence worth reading

- **Cognitive complexity:** `evidence.increments` lists each point with its `type`, `line` and
  `nesting`.
- **Cyclomatic complexity:** `evidence.decisions` lists each decision point and its line.
- **Duplication:** each `duplication.token_clone` finding is one copied block, however long; its
  `current` is the number of lines the copies take up together, and `evidence.tokens` is the length
  of the block. `duplication.token_clone_density` gives the share of duplicated lines.

## Matching findings between versions

A finding's identity is its rule plus its location:

- for function rules, the file and the function name;
- for boundary violations, the two files;
- for cycles, the files in the cycle;
- for duplication, the files and functions that hold the copies, not the copied text.

A file that is moved without being edited keeps its findings. A finding looks new if you rename its
function, or if you move and edit its file in the same change.

## Other formats

`lumioguard-cc check --format sarif` prints the active findings as SARIF 2.1.0 for code scanning
tools. It holds less than the JSON report, so read the JSON yourself.
