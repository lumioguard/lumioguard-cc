---
description: What lumioguard CC is, what it checks, and what it does not do.
---

# What is lumioguard CC?

lumioguard CC is a command-line tool that keeps AI-written code clean as a project grows. It sits in
the coding agent's loop, checks every change for problems, and blocks the agent from finishing until
the code is clean. "CC" stands for **Crap Cleaner**.

## The problem it solves

Coding agents write fast, but the code they leave behind gets worse over time. Functions grow long and
complex, blocks get copied instead of shared, and imports tangle across layers. Each change passes
review because the mess is small, but it adds up, and eventually the code is hard to change.

The worse it gets, the more tokens an agent needs to read and rewrite on every future task. lumioguard
CC breaks that cycle by checking every change as it happens, keeping the codebase maintainable and
cheap for agents to work with.

## What it does

<div class="grid cards" markdown>

-   :lucide-gauge:{ .lg .middle .lg-icon } **Measures**

    ---

    Scores every function and file: how complex it is, how long, how much is copied, and how files
    depend on each other.

-   :lucide-git-compare:{ .lg .middle .lg-icon } **Compares**

    ---

    Matches today's results against an earlier version, usually the last Git commit or the `main`
    branch.

-   :lucide-list-checks:{ .lg .middle .lg-icon } **Decides**

    ---

    Fails only when a change added a problem or made one worse. Problems that were already there stay
    visible but do not fail.

</div>

## What it checks

| Check | In plain words | Details |
| --- | --- | --- |
| Complexity | Functions with too many paths, or too hard to follow | [Complexity](../rules/complexity.md) |
| Size | Functions that are too long or take too many inputs, and files too big to read in one go | [Size](../rules/size.md) |
| Duplication | The same block of code pasted in several places | [Duplication](../rules/duplication.md) |
| Dependencies | Files that depend on too many others, or on each other in a loop | [Dependencies](../rules/dependencies.md) |
| Architecture boundaries | Code in one layer reaching into a layer it should not use | [Boundaries](../rules/boundaries.md) |
| Test coverage | Code your tests do not run, read from your test tool's report | [Coverage](../rules/coverage.md) |

It works on **JavaScript, TypeScript, Python 3, Java** up to version 17 **and Go**, including projects
that mix them.

## What it does not do

- **It does not run your code or your tests.** It reads source files.
- **It does not send your code anywhere.** It works offline and never calls an AI model.
- **It does not judge whether code is correct or secure.** A passing check means nothing got harder to
  maintain by the measures above. See [what a check does not prove](../reference/languages.md#what-a-check-does-not-prove).
- **It does not give one overall score.** Complexity, duplication and dependencies measure different
  things, and a single number would hide the real problem.

## Next steps

<div class="grid cards" markdown>

-   [:lucide-download:{ .lg-icon } **Install**](install.md)

    Download the binary for your platform.

-   [:lucide-rocket:{ .lg-icon } **Quick start**](quick-start.md)

    Run your first check in five minutes.

-   [:lucide-book-open:{ .lg-icon } **Key ideas**](key-ideas.md)

    Findings, blocking and comparisons, in plain English.

</div>
