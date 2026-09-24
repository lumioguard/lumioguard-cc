---
description: Common questions about lumioguard CC.
---

# FAQ

## About the tool

??? question "What does 'CC' stand for?"
    **Crap Cleaner.** The tool helps teams find code that has become hard to maintain, and keep new code
    from getting there.

??? question "Does it upload my code or use AI?"
    No. It runs entirely on your machine, needs no account and no network, and never calls an AI model.
    It reads source files and prints results.

??? question "Does it run my code or my tests?"
    No. It only reads source files. For test coverage it reads the report your test tool already
    wrote.

??? question "Which languages does it support?"
    JavaScript, TypeScript, Python 3, Java up to 17, Go, C and C++, including projects that mix them. See
    [Languages and limits](../reference/languages.md).

??? question "Do I need Node.js, Python, Java or Go installed?"
    No. Everything is inside one program. You need Git only to compare with earlier versions.

??? question "Is it open source?"
    Yes, under the [MIT License](https://github.com/lumioguard/lumioguard-cc/blob/main/LICENSE). The
    parsers it includes are open source too; their notices ship with every release.

## Results

??? question "If the check passes, is my code good?"
    Not necessarily. A pass means nothing got harder to maintain by the measures it checks. It does not
    prove the code is correct, secure or well tested. See
    [what a check does not prove](../reference/languages.md#what-a-check-does-not-prove).

??? question "What does 'Source tokens' in the summary mean?"
    How much code there is to read, counted in lexical tokens without comments, and how much a change
    added or removed. It is the closest measure to what a coding agent must take in on every task.
    See [File tokens](../rules/size.md#file-tokens).

??? question "Why does it not give one overall score?"
    Complexity, duplication and dependencies measure different things in different units. Any way of
    adding them up would be arbitrary, would hide the real problem, and would invite people to game the
    score.

??? question "Our codebase already has hundreds of findings. Is it useless for us?"
    No, that is the case it was built for. Compare with Git (`--base`) and only new or worse problems
    can fail. Old problems stay visible so you can clean them up when you choose. See
    [Check your changes](../guides/check-changes.md).

??? question "Are the default thresholds right for my team?"
    They are a reasonable starting point, not a standard. Adjust them in `.lumioguard-cc.json`, and
    review those changes like code.

??? question "Why are the numbers different from SonarQube or ESLint?"
    Tools count complexity differently, for example whether `&&` and `||` add to cyclomatic complexity.
    The cognitive complexity rule follows the published specification, but is not SonarQube's code, so
    edge cases can differ. Compare numbers from one tool only.

## Teams and agents

??? question "Can a coding agent cheat by changing the settings?"
    An agent with write access could lower a threshold or exclude a file. The skill and the hook tell
    it never to, but real protection comes from reviewing changes to `.lumioguard-cc.json` and running
    the same check in [CI](../guides/ci.md).

??? question "Can the Claude Code hook get an agent stuck in a loop?"
    No. After two blocked attempts in a session, the hook lets the agent stop and asks a person to
    review.

??? question "Do we need a stored baseline?"
    Usually not. Comparing with Git needs no files. Use a [stored baseline](../guides/baselines.md) only
    when you want a fixed limit that changes only through review.
