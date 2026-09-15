# Security policy


## Reporting a vulnerability

Report security problems privately. Do not open a public issue.

1. Open the repository's **Security** tab.
2. Choose **Report a vulnerability**.
3. Describe the problem, the version, and the steps to reproduce it.

Only the maintainers can see the report. They will acknowledge it, work with you on a fix, and
credit you in the advisory if you wish.

While the project is below 1.0, only the latest release receives security fixes.

## What counts

The tool reads untrusted code, often in automated pipelines and agent loops. A vulnerability breaks
one of these properties:

- **It stays inside the analyzed root**, including through symbolic links.
- **`--base` extraction is contained.** No `git archive` entry may escape the temporary directory.
- **It never runs code** from the analyzed repository.
- **`check` never writes.** Only `init` and `baseline create` write files.
- **JSON output is one document.** `--format json` prints exactly one JSON document on stdout.
- **The Claude Code Stop hook stays bounded.** It never blocks forever or reports an unmeasured pass.

## What does not count

- A wrong measurement. Open an ordinary issue; rules are defined in
  [the rules documentation](.documentations/rules/index.md).
- Vulnerabilities in the code being analyzed.
- Dependency advisories that `govulncheck` already reports. Open an ordinary issue.

Run `make audit` to check the linked modules and the copied upstreams yourself.
