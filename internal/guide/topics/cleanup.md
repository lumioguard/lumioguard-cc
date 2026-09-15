# Clean up a project

Turn findings into small refactors that keep behaviour, and report the result with numbers taken
from real checks. Everything in `lumioguard-cc guide check` under "Never do these" applies here too.

## 1. Prepare

- Check `git status`. If some changes are not yours, ask whether to commit them, stash them or
  stop. Suggest a branch.
- Note the starting commit with `START=$(git rev-parse HEAD)`. Every later check compares with it.
- Find the project's test, build, type-check and lint commands in `package.json`,
  `pyproject.toml`, `tox.ini`, `Makefile`, `pom.xml`, `build.gradle` or the CI files, and run them
  once.
- If there are no tests, build a comparison before you touch any code, and tell the user.

### No tests: compare old and new

1. Check out the starting commit in a temp folder with `git worktree add "$TMP/lg-start" "$START"`.
   Remove it with `git worktree remove` at the end.
2. In the temp folder, write a small script that calls the public entry points of both copies with
   the same inputs: generated combinations of valid values, edge values such as `0`, empty strings
   and exact thresholds, and invalid input.
3. Compare the return values, thrown errors and effects you can observe, such as saved records or
   sent messages. Run each copy in its own process, so that module state stays separate.
4. Break one line in a scratch copy and confirm the script reports it, so you know it can fail.
5. Run the script after every batch. When you finish, delete it or offer it to the user as a
   starting point for tests.

## 2. Take the inventory

```bash
lumioguard-cc check --format json > "${TMPDIR:-/tmp}/lg-before.json"
```

Keep this file outside the project.

- Group the `findings` by `scope.file` and `scope.symbol`. A function that appears under several
  rules is a hotspot.
- Compare `current` with `threshold` to see how far over each finding is.
- `policy.blockingFindings` is the headline number.

Exit code 2 means some code was not analyzed. Fix that first.

## 3. Order the work

1. Incomplete analysis, because code that was not analyzed is invisible.
2. Dependency cycles and boundary violations.
3. Hotspot functions. Fixing one often clears several findings.
4. Duplicated blocks.
5. The remaining long functions, long parameter lists and high fan-out.

If the scope is large, agree it with the user. Leave generated and vendored code alone, and suggest
excluding it.

## 4. Work in batches

A batch is one hotspot, one cycle or one duplicated block.

1. **Refactor without changing behaviour.** Keep public names, signatures and exports unless the
   user agreed to change them. If a fix has to move or remove one, say so in the report.
2. **Run the tests or the comparison.** If they fail, fix the refactor or revert the batch. Never
   change a test's expectations.
3. **Check against the start** with `lumioguard-cc check --base "$START" --format json`. The batch
   is done when no finding is `new` or `worsened` and more findings are `resolved` than before.
4. **Commit the batch** if the user wants commits.

Stop when the scope is done, when what remains needs a design decision from the user, or when
changes stop reducing findings.

## 5. Report

Take the numbers from `lg-before.json` and from a final check, never from memory:

| | Before | After |
| --- | --- | --- |
| Blocking findings | | |
| Advisory findings | | |
| Worst cognitive complexity | | |
| Duplicated lines | | |
| Dependency cycles | | |
| Source tokens (`size.total_tokens`) | | |

Then list:

- what changed, one line per batch, naming the files;
- which tests or comparisons ran;
- what you left alone and why;
- next steps, such as setting `"block": true` for rules that are now clean.

## Techniques

**Hotspot functions.** A function that is long, nested and complex at once usually does several
jobs in a row, such as validate, compute, save and notify.
1. Name each phase and extract it into its own function.
2. Leave the original as a short coordinator that calls the phases.
3. Work on any phase that still breaks a threshold.

**Cyclomatic complexity** means too many decisions.
- Chains of `if`/`else` or `switch` that map a value to a result: use a lookup table.
- Branches that choose a behaviour: write one small function per behaviour and choose it once.
- Many validation conditions: keep a list of rules and check them in a loop.
- Long boolean expressions: every `&&` and `||` counts, so name the parts.

**Cognitive complexity and nesting depth** mean the code is hard to follow.
- Use guard clauses: handle invalid or trivial cases first and return.
- Move loop bodies into functions.
- Do not write `else` after `return`, `continue` or `throw`.
- `evidence.increments` lists each point with its line and nesting. Start with the lines that have
  the deepest nesting.

**Long functions.**
- Extract cohesive steps.
- Move long literal tables to module level.
- Do not split in the middle of a step just to get under the limit.

**Long parameter lists.** Group parameters that travel together into one object. For a public
function, keep the old signature as a thin wrapper, or ask the user before changing it.

**Duplicated blocks.** The finding's evidence lists every copy.
- If the copies do the same job, extract one function into a module that both callers already
  depend on. Then check that this adds no cycle or boundary violation.
- If the copies only look alike, leave them and say why.

**Dependency cycles.**
- Move the shared types to a third module that neither file imports back.
- Or define an interface in the lower layer and pass the implementation in.
- In Python, import the modules afterwards to confirm nothing breaks at run time.

**Boundary violations.**
- Move the code to a layer that is allowed to depend on it.
- Or declare the port in the inner layer, such as a repository interface, and implement it
  outside.
- Do not widen `mayDependOn`: the boundaries are the team's decision.

**High fan-out.** Split the file by responsibility, or move the coordinating code up to where the
application is wired together.

**Large files.** Split by responsibility, and move large literal tables into their own file. Do not
spread one function over several files. A refactor should not grow `size.total_tokens` much; if it
does, look for copied code.
