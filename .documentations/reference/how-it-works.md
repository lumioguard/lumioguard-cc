---
description: The steps of a check, from reading files to the exit code.
---

# How it works

## The steps of a check

1. **Find files.** Walk the project in a fixed order, and keep files that match `include` and no
   `exclude` pattern.
2. **Parse.** Each file goes to the parser for its language. Files are parsed in parallel, but results
   always come out in the same order.
3. **Translate.** Every language is translated into one shared model of functions and their control
   flow. This is why a value of 7 means the same in TypeScript, Python, Java and Go.
4. **Measure functions.** Complexity and size are computed once, from the shared model.
5. **Measure across files.** Token streams are searched for copies, imports are traced into a
   dependency graph, and an LCOV report is read if one is configured.
6. **Apply thresholds.** A measurement above its threshold becomes a finding.
7. **Compare.** With `--base` or `--baseline`, every finding is labelled new, worsened, existing or
   resolved.
8. **Decide.** The result is incomplete if anything required was not analyzed. Otherwise it fails if a
   blocking finding is new or worse. Otherwise it passes.

## Comparing with Git

With `--base REF`, the tool:

1. finds the merge base of `HEAD` and `REF`;
2. exports that commit with `git archive` to a temporary folder, without touching your checkout or index;
3. analyzes it with today's configuration;
4. compares the two results, then deletes the temporary folder.

If Git is missing, the folder is not a Git repository, or the reference does not exist, the check stops
with exit code 2.

## Same input, same output

Results are sorted with fixed keys, so the same code and configuration always produce the same
measurements, findings and diagnostics, in the same order. Only the `run` block of the JSON report,
with its ID, time and duration, changes between runs. This makes it safe to compare runs and to store
results.
