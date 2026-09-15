---
description: Every lumioguard-cc command and flag, with examples.
---

# Commands

<div class="lg-summary" markdown>

| Command | Does |
| --- | --- |
| [`init`](#init) | Creates `.lumioguard-cc.json` with the defaults |
| [`check`](#check) | Analyzes the code and decides pass, fail or incomplete |
| [`baseline create`](#baseline-create) | Saves today's results as a stored baseline |
| [`explain`](#explain) | Prints the exact definition of a rule |
| [`guide`](#guide) | Prints step-by-step guides |
| [`doctor`](#doctor) | Shows the setup: configuration, languages and Git |
| [`hook claude-stop`](#hook-claude-stop) | The Claude Code Stop hook |
| [`version`](#version) | Prints the version |

</div>

## Global flags

These work with every command.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--root PATH` | The current folder | The project folder to work on |
| `--format human\|json\|sarif` | `human` | `json` prints one JSON document on standard output; `sarif` (with `check` only) prints the findings as SARIF 2.1.0 |
| `-h`, `--help` | | Help for any command |

## init

```bash
lumioguard-cc init
```

Writes `.lumioguard-cc.json` with the [default configuration](configuration.md). It never overwrites
an existing file: if one exists, it stops with exit code 2.

## check

```bash
lumioguard-cc check                            # the whole project
lumioguard-cc check --base HEAD                # what uncommitted work made worse
lumioguard-cc check --base main                # what the branch made worse
lumioguard-cc check --baseline initial         # compare with a stored baseline
lumioguard-cc check --base HEAD --format json  # full report for tools and agents
lumioguard-cc check --base main --format sarif # findings for code scanning tools
```

| Flag | Meaning |
| --- | --- |
| `--base REF` | Compare with a Git commit, branch or tag. The comparison point is where `HEAD` and `REF` meet (the merge base). |
| `--baseline NAME` | Compare with the stored baseline `.lumioguard-cc/baselines/NAME.json` |

Use either `--base` or `--baseline`, not both. `check` never changes files, configuration or Git state.

**Exit codes:** 0 passed, 1 failed, 2 incomplete or invalid input. They are the same for every format.
See [Results and report](results.md).

## baseline create

```bash
lumioguard-cc baseline create --name initial
lumioguard-cc baseline create --name initial --replace
```

| Flag | Meaning |
| --- | --- |
| `--name NAME` | Required. Letters, numbers, dots, underscores and hyphens. |
| `--replace` | Overwrite a baseline with the same name |

Saves the results to `.lumioguard-cc/baselines/NAME.json`. No file is written if part of the code could
not be analyzed. See [Stored baselines](../guides/baselines.md).

## explain

```bash
lumioguard-cc explain complexity.cognitive
```

```text
Cognitive complexity (vendor-defined)
Sonar's measure of how difficult control flow is to understand: structural increments plus nesting penalties.
Variant: Independent Go implementation of the published Cognitive Complexity specification ...
Limitations: This is not SonarJS output. ...
Source: https://www.sonarsource.com/resources/cognitive-complexity/
```

Prints a rule's definition, exact counting variant, limitations and source. The IDs are listed in
[Rules](../rules/index.md#all-checks). An unknown ID exits with code 2.

## guide

```bash
lumioguard-cc guide            # list the guides
lumioguard-cc guide check      # print one
```

Prints short task guides written for people and coding agents: `setup`, `check`, `cleanup`, `report`
and `config`. They match the installed version.

## doctor

```bash
lumioguard-cc doctor
```

```text
Environment OK
Go go1.27.1 windows/amd64
Configuration: /path/to/project/.lumioguard-cc.json (loaded)
6 source files detected
Adapter javascript-typescript 1.0.0: available
Adapter python 1.0.0: available
Adapter java 1.0.0: available
Git: available (git version 2.46.0.windows.1)
```

Shows whether the configuration was found, how many source files match, which languages are available
and whether Git works. Useful when a check does not behave as expected.

## hook claude-stop

```json
{ "type": "command", "command": "lumioguard-cc hook claude-stop", "timeout": 60 }
```

The [Claude Code](../guides/coding-agents.md) Stop hook. It reads Claude Code's hook input on standard
input, runs a check for the project, and prints a decision that stops Claude from finishing when the
check fails. After two failed attempts in a session it stops blocking and asks a person to review.

| Environment variable | Meaning |
| --- | --- |
| *(none)* | Compare with `HEAD` |
| `LUMIOGUARD_CC_BASE` | Compare with this Git reference instead |
| `LUMIOGUARD_CC_BASELINE` | Compare with this stored baseline instead |

## version

```bash
lumioguard-cc version
```

Prints the version number. `lumioguard-cc --version` does the same.
