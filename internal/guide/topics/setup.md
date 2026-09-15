# Set up a project

Goal: a reviewed `.lumioguard-cc.json`, the current numbers reported to the user, and, if the user
wants it, agents and CI that check their own changes.

1. **Work from the repository root** (`git rev-parse --show-toplevel`). Supported languages:
   JavaScript, TypeScript, Python 3, Java up to 17 and Go. If the project has none, stop.

2. **Create the configuration:**

   ```bash
   lumioguard-cc init
   ```

   It never overwrites an existing file. Exclude generated or vendored code. If the folders show
   clear layers, propose architecture boundaries, and add them only after the user agrees, because
   a new violation fails the check. See `lumioguard-cc guide config`.

3. **Ignore the cache:** add `.lumioguard-cc/cache/` to `.gitignore`. Commit `.lumioguard-cc.json`.

4. **See where the code stands:**

   ```bash
   lumioguard-cc check --format json
   ```

   Group the findings by file and rule, and name the worst functions. Exit code 2 means some code
   was not analyzed; fix that first (see `lumioguard-cc guide check`). Findings in existing code
   are normal, and a comparison never blocks on them.

5. **Decide what blocks.** Only rules with `"block": true` fail a check, stop an agent or fail CI.
   `init` makes every metric rule a warning, so by default only dependency cycles and boundary
   violations block, and a new function with cognitive complexity 35 still passes. Offer to set
   `"block": true` on the metric rules, keeping the thresholds. This is safe on old code, because a
   comparison blocks only findings that a change made new or worse.

6. **Connect agents, if the user wants it.**

   **Claude Code:** merge this into `.claude/settings.json`, keeping any existing hooks:

   ```json
   { "hooks": { "Stop": [ { "hooks": [ { "type": "command", "command": "lumioguard-cc hook claude-stop", "timeout": 60 } ] } ] } }
   ```

   If the binary is not on `PATH`, put its full path in `command`. On Windows, use forward
   slashes, and quote the path if it contains spaces. When Claude tries to finish, the hook
   compares with `HEAD` and blocks, listing what the session made worse. After two blocked attempts
   it lets Claude stop and asks a person to review. To compare with something else, set
   `LUMIOGUARD_CC_BASE` to a Git reference, or `LUMIOGUARD_CC_BASELINE` to a stored baseline.
   To confirm the hook runs, run this from the project root; no output means nothing blocks:

   ```bash
   echo '{"session_id":"setup-test","cwd":"."}' | lumioguard-cc hook claude-stop
   ```

   **Codex and other agents:** add this to `AGENTS.md` or the agent's rules file:

   ```text
   Before finishing a code change, run `lumioguard-cc check --base HEAD --format json`.
   Exit 0 means done. Exit 1 means fix the findings classified new or worsened, then run it again.
   Exit 2 means the analysis is incomplete; read the diagnostics. Never lower thresholds, add
   exclusions or replace a baseline to pass. Details: `lumioguard-cc guide check`.
   ```

7. **Add CI, if the user wants it.** This fails a pull request only on what the branch made worse
   (GitHub Actions). The action installs the release named by its tag, so the tag pins the version:

   ```yaml
   - uses: actions/checkout@v4
     with:
       fetch-depth: 0            # the comparison needs the target branch
   - uses: lumiostack/lumioguard-cc@v{{version}}
   ```

   To show findings on the pull request diff, add `with: { sarif-file: lumioguard-cc.sarif }` and
   upload the file with `github/codeql-action/upload-sarif@v3` under `if: always()`. For other CI
   systems, download the release archive, verify it against `SHA256SUMS`, and run
   `lumioguard-cc check --base origin/main`.

8. **Report:** the version and where it is installed, what you configured and why, the numbers
   and the worst functions, the integrations you added, and what the user still needs to decide.
