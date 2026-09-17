# lumioguard CC agent skill

`lumioguard-cc` is a small agent skill for Claude Code, Codex and other coding agents. It does
three things:

1. installs the lumioguard CC CLI;
2. points the agent to the CLI's built-in guides (`lumioguard-cc guide`);
3. states the rules: do not change thresholds, exclusions or baselines to make a check pass, and ask
   the user before adding boundaries, making rules blocking, or adding hooks, CI or `PATH` changes.

The detailed instructions live in the CLI, not in the skill, so they match the installed version and
the skill stays short.

## Install it

Give users this prompt to paste into their coding agent:

```text
Install lumioguard CC in this project.

1. Add the agent skill: npx skills add https://github.com/lumioguard/lumioguard-cc -y
2. Using that skill, install the lumioguard-cc CLI from
   https://github.com/lumioguard/lumioguard-cc/releases/latest and verify the download.
3. Set up this project with the skill, show me where the code stands, and ask me before
   adding architecture boundaries, agent hooks or PATH changes.

From now on, check your own code changes with lumioguard-cc before you finish.
```

To add only the skill, run `npx skills add https://github.com/lumioguard/lumioguard-cc -y`.
`-y` skips questions, which would otherwise stall an agent. The skill installs into
`.agents/skills/lumioguard-cc/` and is linked for Claude Code. Add `-g` to install it for every
project.

## Files

| File | Contents |
| --- | --- |
| `lumioguard-cc/SKILL.md` | Install check, which guide to read, the change loop and the rules |
| `lumioguard-cc/references/install.md` | Downloading, verifying and installing a release |
| `evals/evals.json` | Test prompts for checking the skill after changes |

## Keeping it accurate

- **Commands, flags, the report or configuration change:** update `internal/guide/topics/`, not
  the skill. Tests keep the guides in step with the CLI.
- **Release file names change:** update `references/install.md` in the same change. The names are
  set in `.github/workflows/release.yml`.
- **The skill or the CLI is hosted elsewhere:** change the URLs in the prompt above and in
  `references/install.md`.

After changing the skill, run the prompts in `evals/evals.json` on a sample project, with and
without the skill, and check the expectations listed with each prompt.
