## Technical-debt measurement loop

This repository uses the deterministic `lumioguard-cc` CLI (lumioguard CC, "Crap Cleaner"). Before finishing a material change, run:

```bash
lumioguard-cc check --base HEAD --format json
```

`--base HEAD` compares the working tree with the last commit, so the repository's existing debt stays visible in the report but only what you introduced or made worse can fail the gate. Nothing has to be created or kept up to date for this to work. Use `--base main` instead when the whole branch is under review, or `--baseline <name>` when the task tells you a reviewed baseline applies.

Exit code 0 means the configured gate passed, 1 means the analysis completed with new or worsened blocking findings, and 2 means required analysis was incomplete or invalid. Inspect the evidence on findings classified `new` or `worsened`, make justified code changes, and re-run the same command with the same comparison flag. Numbers are only comparable between runs that used the same reference. Do not weaken policies, add exclusions, or replace the baseline solely to make a check pass. Stop after two unsuccessful correction attempts and report the remaining evidence to the user.

This is a portable command-based convention. It is not a claim that every Codex host exposes an automatic lifecycle hook.
