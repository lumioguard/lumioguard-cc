---
description: Save a snapshot of today's results and compare with it until the team reviews a new one.
---

# Stored baselines

A stored baseline is a saved snapshot of today's results. Later checks compare with it, so the code
can get better but not worse, until someone reviews and saves a new snapshot.

<div class="lg-summary" markdown>

**Most teams do not need one.** Comparing with Git (`--base`) needs no files and no upkeep. Use a
stored baseline when you want a fixed limit that moves only through review, or when there is no Git
history to compare with.

</div>

## Which comparison to use

| Situation | Use |
| --- | --- |
| You or a coding agent are working on a change | `--base HEAD` |
| Checking a pull request | `--base main`, or whatever the target branch is |
| A fixed debt limit that changes only when reviewed | A stored baseline |
| A first look at a codebase | No comparison: `lumioguard-cc check` |

## Create and use a baseline

```bash
lumioguard-cc baseline create --name initial     # save today's results
lumioguard-cc check --baseline initial           # compare with them later
```

- The snapshot is saved as `.lumioguard-cc/baselines/initial.json`. Names can use letters, numbers,
  dots, underscores and hyphens.
- An existing baseline is never overwritten unless you add `--replace`.
- **Commit the file.** CI needs the same snapshot, and a new snapshot should show up in review. A
  baseline kept on one machine lets anyone make a failing check pass quietly.
- A baseline is not created if part of the code could not be analyzed.

## When a baseline stops matching

A baseline is used only while the settings it was made with still apply. Otherwise the comparison is
refused and the result is incomplete (exit code 2):

| Diagnostic | Cause | Fix |
| --- | --- | --- |
| `baseline.config_incompatible` | `.lumioguard-cc.json` changed | Review the change, then `baseline create --name NAME --replace` |
| `baseline.analyzers_incompatible` | A new version counts something differently | Create a replacement baseline |
| `baseline.schema_incompatible` | The baseline file format changed | Create a replacement baseline |

!!! tip
    With `--base`, both sides always use today's settings, so this never happens.

## How findings are matched

A finding is matched by its rule and location: the file and function name for complexity and size, the
file for fan-out, both files for a boundary violation, the files in a cycle, and the copied code itself
for duplication.

Moving a file without editing it keeps its findings matched. Renaming a function, or moving and editing
a file in one change, makes its findings look `new` and the old ones `resolved`.
