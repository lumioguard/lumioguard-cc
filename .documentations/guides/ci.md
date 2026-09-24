---
description: Run lumioguard CC on pull requests with GitHub Actions or any other CI system.
---

# Pull requests and CI

Run the check on every pull request, so a pull request fails only when it adds a problem or makes one
worse.

<div class="lg-summary" markdown>

**In short:** check out the full Git history, then use the GitHub Action, or install a pinned version
and run `lumioguard-cc check --base origin/<target branch>`. The step fails when the exit code is
not 0.

</div>

## GitHub Actions

The action installs the release that matches its tag, so the `uses` line pins the version. Save this
as `.github/workflows/lumioguard-cc.yml`:

```yaml
name: lumioguard CC

on:
  pull_request:

permissions:
  contents: read

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0            # the comparison needs the target branch

      - uses: lumioguard/lumioguard-cc@v0.2.0
```

On a pull request the action compares with the target branch. The summary appears in the job log, and
the job fails when a blocking finding is new or worse.

| Input | Default | Meaning |
| --- | --- | --- |
| `version` | The action's tag | Release to install, such as `0.2.0` |
| `base` | The pull request's target branch | Git reference to compare with. Empty on other events, which checks the whole project. |
| `working-directory` | `.` | Folder that holds `.lumioguard-cc.json` |
| `sarif-file` | none | Write the findings as SARIF to this path instead of printing the summary |
| `args` | none | Extra arguments for `lumioguard-cc check` |

The action's `exit-code` output holds 0, 1 or 2, so a later step can act on it.

### Code scanning with SARIF

To see each finding on the pull request diff and in the repository's **Security** tab, write a SARIF
file and upload it. The upload step needs `security-events: write` and `if: always()`, because the check
step fails the job when a finding blocks:

```yaml
permissions:
  contents: read
  security-events: write

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: lumioguard/lumioguard-cc@v0.2.0
        with:
          sarif-file: lumioguard-cc.sarif

      - uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: lumioguard-cc.sarif
```

Code scanning matches findings between runs by their IDs, so a pull request shows only the findings it
adds. [SARIF output](../reference/results.md#sarif-output) lists what the file holds.

### Without the action

Any workflow can install a release itself. Verify the download against `SHA256SUMS`, and pin the
version so that an upgrade never changes results unnoticed:

```yaml
      - name: Install lumioguard CC
        env:
          VERSION: 0.2.0
        run: |
          NAME="lumioguard-cc_${VERSION}_linux_amd64"
          BASE="https://github.com/lumioguard/lumioguard-cc/releases/download/v$VERSION"
          curl -fsSLO "$BASE/$NAME.tar.gz"
          curl -fsSLO "$BASE/SHA256SUMS"
          grep " $NAME.tar.gz\$" SHA256SUMS | sha256sum -c -
          tar -xzf "$NAME.tar.gz"
          echo "$PWD/$NAME" >> "$GITHUB_PATH"

      - name: Check what this pull request changed
        env:
          TARGET: ${{ github.base_ref }}
        run: lumioguard-cc check --base "origin/$TARGET"
```

!!! tip "Keep the full report"
    Add `--format json > lumioguard-cc-report.json` to the check, then upload the file with
    `actions/upload-artifact`. The report lists every finding and measurement, not just the first ten.

## Other CI systems

Any CI works the same way:

1. **Fetch enough history** to include the target branch. A shallow clone cannot find the merge base.
2. **Install a pinned version** and verify it against `SHA256SUMS`, as in [Install](../get-started/install.md).
3. **Run** `lumioguard-cc check --base <target branch>` and let the exit code decide the job. Add
   `--format sarif` if the CI system reads SARIF:

| Exit code | Result | CI outcome |
| --- | --- | --- |
| 0 | Passed | Success |
| 1 | A blocking finding is new or worse | Fail |
| 2 | Incomplete analysis, bad input or configuration | Fail, and fix the cause |

## Choose what fails the build

Only rules with `"block": true` can fail a pull request. The defaults make size and complexity rules
advisory, so start by making blocking the rules your team agrees on. See
[Configuration](../reference/configuration.md#metrics).

!!! note "Protect the settings too"
    A pull request can lower a threshold or add an exclusion to pass. Ask reviewers to treat changes to
    `.lumioguard-cc.json` like code changes, for example with a `CODEOWNERS` entry.
