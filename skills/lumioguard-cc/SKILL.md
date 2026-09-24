---
name: lumioguard-cc
description: Install and use lumioguard CC (the lumioguard-cc CLI), which measures complexity, size, duplication, dependency cycles and layer violations in JavaScript, TypeScript, Python, Java, Go, C and C++, and fails only on what a change made worse. Use it when the user asks to install or set up lumioguard or lumioguard-cc, wants a technical-debt, complexity or architecture check, or asks to clean up, simplify or refactor a codebase; when a repository has .lumioguard-cc.json; when a lumioguard-cc hook blocks you; and before declaring any code change finished in a project that uses it.
compatibility: Needs a shell and Git. Installing needs network access to the GitHub releases page, or Go 1.27 or later.
---

# lumioguard CC

The CLI carries its own instructions for the installed version, so this skill only gets it
installed and points you at the right guide.

## 1. Make sure the CLI is installed

Run `lumioguard-cc --version`. If it is missing, follow [references/install.md](references/install.md).
Ask before changing `PATH`. Until the user agrees, call the binary by its full path.

## 2. Read the guide for the job, then follow it

| Job | Run |
| --- | --- |
| Install or set up lumioguard CC, agent hooks or CI | `lumioguard-cc guide setup` |
| Finish a code change, or respond to a blocking hook | `lumioguard-cc guide check` |
| Clean up, simplify or refactor existing code | `lumioguard-cc guide cleanup`, then `lumioguard-cc worklist` for where to start |
| Read the JSON report | `lumioguard-cc guide report` |
| Edit `.lumioguard-cc.json` | `lumioguard-cc guide config` |

## The loop for any code change

```bash
lumioguard-cc check --base HEAD --format json
```

- **Exit 0:** done. If some findings are classified `new` or `worsened` but not blocking, fix them
  when reasonable.
- **Exit 1:** fix the findings classified `new` or `worsened`, then run the check again. After two
  failed attempts, stop and report what remains.
- **Exit 2:** the analysis is incomplete. The check guide explains why.

Read the JSON, not the human summary, which lists only ten findings.

## Never, to make a check pass

- Lower thresholds, or turn off `block` or `enabled`.
- Exclude code you wrote.
- Create or replace baselines.
- Rename code or move files to reset its history.
- Change test expectations.

If a rule seems wrong, tell the user.

## Ask the user before

- adding architecture boundaries;
- making rules blocking;
- adding agent hooks or CI;
- changing `PATH`.
