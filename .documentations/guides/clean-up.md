---
description: Reduce existing findings in small steps, and report the numbers before and after.
---

# Clean up existing code

Find the code with the most findings, fix it in small steps without changing what it does, and report
the numbers before and after.

<div class="lg-summary" markdown>

**In short:** save today's results, fix one problem area at a time, compare each step with your
starting commit, and stop when the numbers stop improving.

</div>

## 1. Prepare

- Start from a clean working folder, ideally on a new branch.
- Note the starting commit: `git rev-parse HEAD`.
- Run the project's tests once, so you know they pass before you change anything. If there are no
  tests, write a few for the code you plan to change first.

## 2. Take an inventory

```bash
lumioguard-cc check --format json > lumioguard-before.json
lumioguard-cc worklist
```

Keep the first file outside the project. It is your "before" picture. The [worklist](../reference/commands.md#worklist)
orders the places to fix: cycles and boundary violations first, then **hotspots**, functions that break
several rules at once, such as long, deeply nested and complex, then duplicated blocks. Fixing one
hotspot often clears several findings. `--path` narrows it to one folder and `--rule` to one rule.

## 3. Work in order

1. **Incomplete analysis first.** Code that could not be read is invisible to every other rule.
2. **Dependency cycles and boundary violations.** They make every other change harder.
3. **Hotspot functions.**
4. **Duplicated blocks.**
5. **The remaining long functions, long parameter lists and busy files.**

## 4. Fix one area at a time

For each area:

1. Refactor without changing behaviour. Keep public names and signatures unless you agree otherwise
   with your team.
2. Run the tests.
3. Compare with the starting commit:

    ```bash
    lumioguard-cc check --base <starting commit>
    ```

    The step is done when nothing is `new` or `worsened`, and more findings are `resolved` than
    before.

4. Commit, naming what you simplified.

Splitting a function into pieces that each still break a threshold is not a fix. Each rule page has
techniques for lowering its number: [complexity](../rules/complexity.md#how-to-lower-it),
[size](../rules/size.md#how-to-lower-it), [duplication](../rules/duplication.md#how-to-fix-it),
[dependencies](../rules/dependencies.md#how-to-fix-a-cycle) and
[boundaries](../rules/boundaries.md#how-to-fix-a-violation).

## 5. Report the result

Build the report from real output, never from memory: `lumioguard-before.json` and a final
`lumioguard-cc check --format json`.

| | Before | After |
| --- | --- | --- |
| Blocking findings | from `policy.blockingFindings` | |
| Advisory findings | from `policy.warningFindings` | |
| Worst cognitive complexity | function and value | |
| Dependency cycles | count | |

Once a rule is clean, set `"block": true` for it so it stays clean.

## Worked examples

The repository has one small project per language, written badly and then cleaned up. Every number in
their reports came from running the tool.

| Project | Blocking findings, before and after |
| --- | --- |
| [TypeScript order service](https://github.com/lumiostack/lumioguard-cc/blob/main/.examples/typescript-order-service/REPORT.md) | 10 → 0 |
| [Python inventory](https://github.com/lumiostack/lumioguard-cc/blob/main/.examples/python-inventory/REPORT.md) | 9 → 0 |
| [Java billing](https://github.com/lumiostack/lumioguard-cc/blob/main/.examples/java-billing/REPORT.md) | 9 → 0 |

!!! tip "Cleaning up with a coding agent"
    Ask the agent to "clean up this project using lumioguard-cc". The installed skill and
    `lumioguard-cc guide cleanup` give it this process, including a way to check that behaviour did not
    change when a project has no tests.
