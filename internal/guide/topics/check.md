# Check a code change

Before you report a change as done, check that it did not make the code harder to maintain.

1. **Pick what to compare with:**
   - `HEAD` for uncommitted work;
   - the commit you started from, if you commit during the task (note `git rev-parse HEAD` first);
   - whatever the project's instructions or CI use, such as `--base main` or `--baseline NAME`.

2. **Run the check:**

   ```bash
   lumioguard-cc check --base HEAD --format json
   ```

   Read the JSON, not the human summary. The summary lists at most ten findings, including old
   ones, so the findings your change caused can be missing from it.

3. **Act on the exit code:**

   | Exit | Meaning | Do this |
   | --- | --- | --- |
   | 0 | Nothing blocking is new or worse | Also look for findings with `"blocking": false` classified `new` or `worsened`. Your change caused them too: fix them if reasonable, otherwise mention them. |
   | 1 | A blocking finding is new or worse | Fix the findings classified `new` or `worsened`, then run the same command again |
   | 2 | Incomplete analysis or invalid input | See [Exit code 2](#exit-code-2) |

4. **After two attempts that still fail,** stop and report what remains, with the findings.

If the folder is not a Git repository, run `lumioguard-cc check --format json` before changing
anything and again afterwards, then compare the findings for the files you touched.

## When the Claude Code hook blocks you

The message lists what your session made worse and the command that reproduces it. Fix those
findings, run that command, then finish. After two blocked attempts the hook lets you stop; report
what still fails and why.

## Fixing findings

Fix the cause, not the number, and keep the behaviour the same.

| Rule | Usual fix |
| --- | --- |
| `complexity.cyclomatic` | Replace `if` or `switch` chains with a lookup table; extract branches into named functions |
| `complexity.cognitive` | Guard clauses and early returns; extract nested blocks; name parts of long conditions |
| `complexity.nesting_depth` | Invert conditions and return early; move loop bodies into functions |
| `size.function_lines` | Extract each step into a well-named function |
| `size.parameter_count` | Group parameters that travel together into one object |
| `size.file_tokens` | Split the file by responsibility; move large literal tables out |
| `duplication.token_clone` | Extract one shared function and call it from each copy |
| `coupling.module_fan_out` | Split the file by responsibility |
| `dependency.cycle` | Move shared types to a third module, or invert one import behind an interface |
| `architecture.boundary_violation` | Move the code to an allowed layer, or define an interface in the inner layer |

Splitting a function into pieces that each still break a threshold is not a fix.
`lumioguard-cc explain <rule-id>` prints a rule's exact definition, and each finding's `evidence`
shows where its number comes from. More techniques: `lumioguard-cc guide cleanup`.

## Exit code 2

| Problem | Fix |
| --- | --- |
| `*.parse_failed` | The file is not valid code: fix syntax you wrote. JSX needs `.jsx` or `.tsx`. Only Python 3 is supported. In C and C++, only the first branch of each `#if` is read; K&R definitions and Objective-C are not supported, so exclude those files. |
| `adapter.unsupported_language` | `include` matches a file type the tool cannot read; the user should narrow `include` |
| `baseline.config_incompatible` | The stored baseline was made with another configuration: use `--base` and tell the user; do not replace the baseline |
| `coverage.report_unavailable`, `coverage.report_stale` | Rerun the tests with coverage, or ask whether coverage should be `required` |
| A Git error and no JSON | Not a Git repository, or the reference does not exist: compare before and after as above |

## Never do these to make a check pass

- Lower a threshold, or set `block` or `enabled` to `false`.
- Add an `exclude` for code you wrote or changed.
- Create or replace a stored baseline.
- Rename functions or move files to reset their history.
- Change a test's expectations.

People rely on these numbers to judge code, including yours. If a rule seems wrong for the code,
tell the user and let them decide.
