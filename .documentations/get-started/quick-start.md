---
description: Set up lumioguard CC in a project and run your first checks.
---

# Quick start

Five steps: create the configuration, run a check, compare a change with the last commit, make a rule
block, and fix the finding. You need lumioguard CC [installed](install.md) and a project in Git.

The output on this page is real. It comes from the Python example project in the repository, before
and after adding one deeply nested function.

## 1. Create the configuration

In the root folder of your project, run:

```bash
lumioguard-cc init
```

```text
Created /path/to/project/.lumioguard-cc.json
```

`init` writes the default settings and never overwrites an existing file. Commit
`.lumioguard-cc.json`, and add `.lumioguard-cc/cache/` to your `.gitignore`.

## 2. See where the code stands

```bash
lumioguard-cc check
```

```text
lumioguard CC check: PASSED
10 files analyzed in 214 ms
0 blocking, 0 advisory findings
Source tokens: 1549
No active findings exceeded the configured policy.
```

This project has no findings. Most existing projects report findings on the first check. That is
normal: the next step compares a change instead of the whole project, so existing findings do not
appear. `Source tokens` is how much code there is to read; the next step shows how much a change
adds.

## 3. Check only what you changed

Make a change, then compare it with your last commit:

```bash
lumioguard-cc check --base HEAD
```

```text
lumioguard CC check: PASSED
10 files analyzed in 565 ms
0 blocking, 2 advisory findings
Source tokens: 1720 (+171 since the base commit)
- [warning] inventory/service/reporting.py:27: complexity.cognitive is 26 count; the configured threshold is 15 (new)
- [warning] inventory/service/reporting.py:27: complexity.nesting_depth is 6 count; the configured threshold is 4 (new)
```

The new function is too hard to follow, and both findings are marked `(new)`. The change added 171
tokens of code. The check still passed, because the default settings only **warn**.

## 4. Decide what blocks

To make these rules fail the check, set `"block": true` in `.lumioguard-cc.json`:

```json hl_lines="2 3"
"metrics": {
  "cognitive": { "enabled": true, "threshold": 15, "severity": "warning", "block": true },
  "nestingDepth": { "enabled": true, "threshold": 4, "severity": "warning", "block": true },
```

```text
lumioguard CC check: FAILED
10 files analyzed in 605 ms
2 blocking, 0 advisory findings
Source tokens: 1720 (+171 since the base commit)
- [warning] inventory/service/reporting.py:27: complexity.cognitive is 26 count; the configured threshold is 15 (new)
- [warning] inventory/service/reporting.py:27: complexity.nesting_depth is 6 count; the configured threshold is 4 (new)
```

The command now exits with code 1. Blocking is safe on old code: with `--base`, only findings that are
new or worse can fail.

## 5. Fix and check again

Simplify the function, for example with early returns, and run the same command until it passes. Every
finding names its file and line. To see why a number is what it is:

```bash
lumioguard-cc explain complexity.cognitive
```

## What next

<div class="grid cards" markdown>

-   [:lucide-bot:{ .lg-icon } **Coding agents**](../guides/coding-agents.md)

    Let Claude Code or Codex run this loop for you.

-   [:lucide-git-pull-request:{ .lg-icon } **Pull requests and CI**](../guides/ci.md)

    Fail pull requests that make the code worse.

-   [:lucide-list-checks:{ .lg-icon } **Rules**](../rules/index.md)

    What each check measures, and how to fix it.

-   [:lucide-book-open:{ .lg-icon } **Key ideas**](key-ideas.md)

    Findings, blocking and comparisons explained.

</div>

!!! tip "Guides in your terminal"
    `lumioguard-cc guide` prints short guides for setup, checking changes and cleanup, matched to the
    version you installed.
