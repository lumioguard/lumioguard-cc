---
description: How lumioguard CC finds blocks of code copied in more than one place.
---

# Duplication

Finds blocks of code that appear, exactly, in more than one place. A bug fixed in one copy stays in the
others.

<div class="lg-summary" markdown>

| | |
| --- | --- |
| **IDs** | `duplication.token_clone_density` for the whole project, and one `duplication.token_clone` finding per copied block |
| **Default** | 5 % of lines, does not block |
| **Settings** | `threshold`, `minTokens` (default 50) and `minLines` (default 5) under `metrics.duplicationPercent` |

</div>

## How it counts

1. Each file is read as a stream of tokens, with comments and whitespace removed.
2. A run of tokens counts as a copy when it:
    - is at least `minTokens` tokens long;
    - covers at least `minLines` lines;
    - matches another run exactly, including names;
    - does not overlap the run it matches.
3. The **density** is the number of duplicated lines divided by all non-blank lines analyzed. It is one
   number for the whole project, and each line counts once.
4. When the density is above the threshold, every copied block is also reported with all its locations.

## What it misses

- **Renamed copies.** A copy with one variable renamed is not an exact match.
- **Copies across languages.** A TypeScript block never matches a Python block.
- In JavaScript and TypeScript, a regular expression, a template string or a piece of JSX text is one
  token.
- In Go, the semicolon the compiler inserts at a line end is one token, the same as a written `;`.

Not every copy is a mistake. Two blocks can look alike today and still need to change for different
reasons.

## How to fix it

The finding's `evidence` in the JSON report lists every copy with its file and lines.

- **If the copies do the same job,** extract one shared function. Put it in a file that both callers
  already import, or in a shared module lower in your layers.
- **Check the result** with `lumioguard-cc check`: a new shared module must not create a
  [dependency cycle](dependencies.md#dependency-cycles) or a [boundary violation](boundaries.md).
- **If the copies only look alike,** leave them, and note why.
