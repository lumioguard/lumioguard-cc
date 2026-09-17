---
description: Every field in .lumioguard-cc.json, with its default.
---

# Configuration

All settings live in `.lumioguard-cc.json` in the project's root folder. `lumioguard-cc init` writes it
with the defaults below. Without the file, a check uses the same defaults.

!!! warning "The file is checked strictly"
    An unknown or misspelled field stops the check with exit code 2 and names the field, so a typo
    cannot silently switch a rule off:

    ```text
    lumioguard-cc: invalid .lumioguard-cc.json: metrics.cyclomatic contains unknown fields: treshold
    ```

## Default configuration

```json
{
  "schemaVersion": "1.0",
  "source": {
    "include": ["**/*.js", "**/*.jsx", "**/*.mjs", "**/*.cjs",
                "**/*.ts", "**/*.tsx", "**/*.mts", "**/*.cts",
                "**/*.py", "**/*.pyw", "**/*.java", "**/*.go"],
    "exclude": ["**/node_modules/**", "**/dist/**", "**/build/**", "**/out/**", "**/coverage/**",
                "**/.git/**", "**/.lumioguard-cc/**", "**/*.min.js", "**/*.generated.*",
                "**/.next/**", "**/.nuxt/**", "**/.output/**", "**/.svelte-kit/**", "**/.turbo/**",
                "**/.cache/**", "**/.parcel-cache/**", "**/__pycache__/**", "**/.venv/**",
                "**/venv/**", "**/site-packages/**", "**/target/**", "**/.gradle/**",
                "**/vendor/**", "**/testdata/**"]
  },
  "metrics": {
    "cyclomatic":         { "enabled": true, "threshold": 12,   "severity": "warning", "block": false },
    "cognitive":          { "enabled": true, "threshold": 15,   "severity": "warning", "block": false },
    "nestingDepth":       { "enabled": true, "threshold": 4,    "severity": "warning", "block": false },
    "functionLines":      { "enabled": true, "threshold": 80,   "severity": "warning", "block": false },
    "parameterCount":     { "enabled": true, "threshold": 5,    "severity": "warning", "block": false },
    "fileTokens":         { "enabled": true, "threshold": 4000, "severity": "warning", "block": false },
    "duplicationPercent": { "enabled": true, "threshold": 5,    "severity": "warning", "block": false,
                            "minTokens": 50, "minLines": 5 },
    "fanOut":             { "enabled": true, "threshold": 20,   "severity": "warning", "block": false }
  },
  "architecture": { "boundaries": [], "blockCycles": true, "blockViolations": true },
  "coverage": { "lcovFile": null, "required": false }
}
```

## source

Which files are analyzed.

| Field | Meaning |
| --- | --- |
| `include` | Glob patterns, relative to the root. A file must match one. `**` matches any number of folders. |
| `exclude` | Glob patterns. A file that matches one is skipped. |

- `.git`, `node_modules`, `__pycache__` and `.lumioguard-cc` are always skipped, and symbolic links are
  never followed.
- **Only include file types the tool supports.** A matching file it cannot read, such as a `.rb` file,
  makes the result incomplete rather than being silently ignored.
- Exclude code nobody edits by hand, such as generated or vendored code. Do not exclude tests or
  scripts just because they score badly.

## metrics

Every size, complexity and duplication rule has the same four fields. [Rules](../rules/index.md)
explains what each one counts.

| Key | Rule |
| --- | --- |
| `cyclomatic` | [Cyclomatic complexity](../rules/complexity.md#cyclomatic-complexity) |
| `cognitive` | [Cognitive complexity](../rules/complexity.md#cognitive-complexity) |
| `nestingDepth` | [Nesting depth](../rules/complexity.md#nesting-depth) |
| `functionLines` | [Function length](../rules/size.md#function-length) |
| `parameterCount` | [Parameter count](../rules/size.md#parameter-count) |
| `fileTokens` | [File tokens](../rules/size.md#file-tokens) |
| `duplicationPercent` | [Duplication](../rules/duplication.md), with `minTokens` and `minLines` |
| `fanOut` | [Fan-out](../rules/dependencies.md#fan-out-and-fan-in) |

| Field | Meaning |
| --- | --- |
| `enabled` | `false` stops findings; the measurement is still recorded |
| `threshold` | A finding is raised when the value is above this number |
| `severity` | `info`, `warning` or `error`: only the label on the finding |
| `block` | `true` lets a new or worse finding fail the check; `false` only reports it |

`severity` and `block` are separate on purpose. A rule can be labelled `error` and only report while you
clean up, then block once the code is clean.

## architecture

| Field | Default | Meaning |
| --- | --- | --- |
| `boundaries` | `[]` | Your layers and what each may import. See [Architecture boundaries](../rules/boundaries.md). |
| `blockCycles` | `true` | New dependency cycles fail the check |
| `blockViolations` | `true` | New boundary violations fail the check |

## coverage

| Field | Default | Meaning |
| --- | --- | --- |
| `lcovFile` | `null` | Path to an LCOV report, or `null` for no coverage |
| `required` | `false` | `true` turns a missing or out-of-date report into an incomplete result |

See [Test coverage](../rules/coverage.md).

## Changing the file

- **With `--base`**, both sides of the comparison use the current file, so you can change thresholds at
  any time.
- **With a stored baseline**, any change makes the baseline incompatible until you replace it.
- **Review changes like code.** Lowering a threshold or adding an exclusion is the easiest way to make
  a failing check pass without fixing anything.
