---
description: Fixes for incomplete results, parse failures, Git errors and other common problems.
---

# Troubleshooting

Start with `lumioguard-cc doctor`. It shows whether the configuration was found, how many files match,
and whether Git works.

## The result is INCOMPLETE

Part of the code could not be analyzed, so the check refuses to pass. The summary names the cause:

```text
lumioguard CC check: INCOMPLETE
1 files analyzed in 71 ms
0 blocking, 0 advisory findings
- [error] adapter.unsupported_language: No installed language adapter supports this included source file
```

An incomplete result never shows a `Source tokens` total, because not everything was read.

| Diagnostic | Fix |
| --- | --- |
| `typescript.parse_failed`, `python.parse_failed`, `java.parse_failed`, `go.parse_failed` | See [A file cannot be parsed](#a-file-cannot-be-parsed) |
| `adapter.unsupported_language` | `include` matches a file type the tool cannot read. Narrow `source.include` to supported types. |
| `coverage.report_unavailable`, `coverage.report_stale` | Run the tests with coverage again, or set `coverage.required` to `false` |
| `baseline.config_incompatible` and other `baseline.*` | See [The stored baseline no longer matches](#the-stored-baseline-no-longer-matches) |

## A file cannot be parsed

```text
- [error] typescript.parse_failed: Could not parse source: Identifier expected 1003 (line 1)
```

- **A real syntax error:** fix the code at the line shown.
- **JSX in a `.js` or `.ts` file:** JSX is read only in `.jsx` and `.tsx` files. Rename the file.
- **Python 2 code:** only Python 3 is supported. Exclude the file.
- **Java newer than 17:** the Java grammar covers Java 17. Newer syntax can fail to parse.
- **Go:** the message names the line and what the parser expected. Generated files that are not
  valid Go, such as templates with a `.go` extension, need an `exclude` entry.

## The configuration is invalid

```text
lumioguard-cc: invalid .lumioguard-cc.json: metrics.cyclomatic contains unknown fields: treshold
```

The file is checked strictly, so a misspelled field stops the check with exit code 2. Fix the field it
names. Compare with the [default configuration](../reference/configuration.md#default-configuration).

## `--base` fails

```text
lumioguard-cc: git rev-parse --verify HEAD^{commit}: fatal: not a git repository (or any of the parent directories): .git
```

- **Not a Git repository:** run the command inside the repository, or pass `--root`.
- **`Needed a single revision`:** the reference does not exist. Check the branch name, and in CI fetch
  it first, such as `origin/main`.
- **No commits yet:** `--base HEAD` needs at least one commit.
- **In CI:** a shallow clone cannot find the merge base. Fetch the full history, such as
  `fetch-depth: 0` in GitHub Actions.

## The stored baseline no longer matches

A stored baseline is compared only while the settings it was made with still apply. If
`.lumioguard-cc.json` changed, or a new version counts differently, review the change and create a
replacement:

```bash
lumioguard-cc baseline create --name initial --replace
```

Or compare with Git instead (`--base`), which always uses today's settings.

## The check or the hook never fails

The default settings make the complexity, size, duplication and fan-out rules **advisory**: they
report findings but never fail. Set `"block": true` on the rules you want enforced. See [Key ideas](../get-started/key-ideas.md#blocking-and-advisory-findings).

## A finding shows as new after a rename or move

Findings are matched by file and function name. Renaming a function, or moving and editing a file in the
same change, makes its findings look `new` and the old ones `resolved`. Commit a move on its own, then
edit.

## My finding is not in the summary

The summary shows at most ten findings, including old ones. Use `--format json` to see all of them.

## The numbers differ from another tool

Tools count complexity differently. For example, some do not count `&&` and `||`, and some treat
`else if` as flat. Use `lumioguard-cc explain <rule-id>` for this tool's exact rules, and do not mix
numbers from different tools.

## Still stuck?

[Open an issue](https://github.com/lumioguard/lumioguard-cc/issues) with the command you ran, the
output, and the output of `lumioguard-cc doctor`.
