---
description: The check summary, exit codes, the JSON report and diagnostics.
---

# Results and report

## The summary

A plain `lumioguard-cc check` prints a short summary. This is the TypeScript example project before its
clean-up:

```text
lumioguard CC check: FAILED
6 files analyzed in 156 ms
16 blocking, 0 advisory findings
Source tokens: 1660
- [error] src/domain/order.ts:1: domain is not allowed to depend on persistence (new)
- [error] src/api/handler.ts:5: complexity.cognitive is 96 count; the configured threshold is 15 (new)
- [error] src/api/handler.ts:5: complexity.cyclomatic is 52 count; the configured threshold is 10 (new)
- [error] src/api/validation.ts:9: complexity.cyclomatic is 16 count; the configured threshold is 10 (new)
- [error] src/api/handler.ts:5: complexity.nesting_depth is 7 count; the configured threshold is 4 (new)
...
```

Each line reads `[severity] where: what (classification)`. `Source tokens` is the size of the analyzed
code; when the check compares with a commit or a baseline, the line adds what the change did to it, such
as `(+171 since the base commit)`. See [File tokens](../rules/size.md#file-tokens).

!!! warning "At most ten findings are shown"
    Findings that already existed are listed too, so a new one can be missing from the summary. The JSON
    report always has every finding.

## Exit codes

| Result | Exit code | Meaning |
| --- | --- | --- |
| `PASSED` | 0 | Everything required was analyzed, and no blocking finding is new or worse |
| `FAILED` | 1 | The analysis finished, and a blocking finding is new or worse |
| `INCOMPLETE` | 2 | Something required could not be analyzed |

Exit code 2 also covers a bad command line, an invalid configuration and internal errors.
**Incomplete always wins over failed**: a check that did not see all the code must not look like a pass
or an ordinary failure.

## Classifications

| Classification | Meaning | Can fail the check |
| --- | --- | --- |
| `new` | Not in the reference | Yes, if the rule blocks |
| `worsened` | In the reference, and the value went up | Yes, if the rule blocks |
| `existing` | In the reference, and the value did not go up | No |
| `resolved` | In the reference, gone now | No |

A finding that improved but is still over its threshold stays `existing`. Without a comparison, every
finding is `new`.

## The JSON report

`lumioguard-cc check --format json` prints exactly one JSON document on standard output. Errors go to
standard error. The same code and configuration always give the same report, apart from the `run`
block. The format is defined by
[`schemas/report.schema.json`](https://github.com/lumiostack/lumioguard-cc/blob/main/schemas/report.schema.json).
It is the complete record; the SARIF output below is derived from it.

| Field | Contents |
| --- | --- |
| `policy` | `status` (`passed`, `failed`, `incomplete`), `blockingFindings`, `warningFindings` |
| `findings` | Every finding, including `resolved` ones |
| `measurements` | Every value recorded, including values under their threshold |
| `diagnostics` | Problems with the analysis itself, not with the code |
| `comparison` | `mode` (`none`, `git`, `baseline`), `reference`, and whether both sides were `comparable` |
| `scope`, `exclusions` | Files found and analyzed per language, and what was skipped |
| `snapshot`, `adapters` | Hashes of the analyzed files and configuration, and analyzer versions |
| `tool`, `run`, `schemaVersion` | Tool version, run ID, time and duration |

### A finding

A real finding from the same project, without its `evidence`:

```json
{
  "id": "9e8afced20d85b247e28",
  "ruleId": "complexity.nesting_depth",
  "title": "complexity.nesting_depth exceeds project threshold",
  "severity": "error",
  "blocking": true,
  "scope": {
    "kind": "function",
    "file": "src/api/handler.ts",
    "symbol": "processOrder",
    "line": 5,
    "endLine": 167,
    "key": "src/api/handler.ts::processOrder"
  },
  "message": "complexity.nesting_depth is 7 count; the configured threshold is 4",
  "classification": "new",
  "current": 7,
  "baseline": null,
  "delta": null,
  "threshold": 4,
  "unit": "count",
  "recheck": "lumioguard-cc check --format json"
}
```

- `blocking` says whether the rule can fail the check. It fails only if the finding is also `new` or
  `worsened`.
- `baseline` and `delta` hold the earlier value and the change when the check compared with something.
- `evidence` explains the number, such as every cognitive complexity point with its line, or the
  location of every copy.

### Measurements

A measurement's `status` is one of:

- `measured`: `value` holds the number;
- `not_applicable`: there was nothing to measure;
- `unavailable`: the number could not be produced.

The last two have a `null` value and a `reason`. A missing number is never shown as 0 or 100 %.

## SARIF output

`lumioguard-cc check --format sarif` prints the same check as one
[SARIF 2.1.0](https://sarifweb.azurewebsites.net/) log, the format that GitHub code scanning, GitLab
and IDE extensions read. Exit codes are unchanged, so a failed check still exits with 1.

- Every active finding becomes a `result` with the rule ID, the message, the file and lines, and the
  classification when the check compared with something. Resolved findings are left out.
- Each `result` carries the finding's ID as a `partialFingerprints` entry, so a code scanning tool can
  tell a finding it has already shown from a new one.
- `level` follows the finding's severity: `error`, `warning` or `note`. Whether a finding blocks is in
  `properties.blocking`, together with `current`, `threshold`, `baseline` and `delta`.
- A duplicated block is placed at its first copy, with the other copies as related locations. A cycle
  is placed at its first file.
- Problems with the analysis appear as `toolExecutionNotifications`, and `executionSuccessful` is
  `false` when the result is incomplete.
- Rules link to their page in this documentation through `helpUri`.

The SARIF log is derived from the JSON report and holds less: no measurements, no snapshot and no
resolved findings. Tools and agents that need those read `--format json`. See
[Pull requests and CI](../guides/ci.md#code-scanning-with-sarif) for uploading the file.

## Diagnostics

Diagnostics are problems with the analysis, not with your code. A diagnostic with `"required": true`
makes the result incomplete.

| Diagnostic | Meaning |
| --- | --- |
| `typescript.parse_failed`, `python.parse_failed`, `java.parse_failed`, `go.parse_failed` | A file is not valid code |
| `adapter.unsupported_language` | A file matched `include`, but no language analyzer supports it |
| `dependency.import_unresolved` | An import looks internal, but matches no file |
| `dependency.external_not_analyzed` | How many imports pointed outside the analyzed code |
| `coverage.report_unavailable`, `coverage.report_stale`, `coverage.no_sources_mapped` | Coverage could not be used |
| `baseline.config_incompatible`, `baseline.analyzers_incompatible`, `baseline.schema_incompatible` | The stored baseline no longer matches |

For fixes, see [Troubleshooting](../help/troubleshooting.md).
