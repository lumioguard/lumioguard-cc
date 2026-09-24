# Worked examples

Five small projects, in TypeScript, Python, Java, C and C++, each written twice: a
deliberately bad `before/` version and a refactored `after/` version. Every
number in the reports was produced by running the built binary against the code
in this directory, not estimated. The Go example is this repository itself; see
the note at the end.

| Example | Language | Problem it models |
| --- | --- | --- |
| [`typescript-order-service`](typescript-order-service/REPORT.md) | TypeScript | One "god function" that validates, prices, taxes, discounts, ships, saves and emails |
| [`python-inventory`](python-inventory/REPORT.md) | Python | A warehouse rebalancing routine with five levels of nesting and a copy-pasted report |
| [`java-billing`](java-billing/REPORT.md) | Java | An invoice calculator whose tax and coupon rules are hard-coded branches |
| [`c-sensor-pipeline`](c-sensor-pipeline/REPORT.md) | C | A sensor batch processor that calibrates, filters, stores and alerts in one function |
| [`cpp-shipping-quotes`](cpp-shipping-quotes/REPORT.md) | C++ | A quote service whose carriers, zones and surcharges are nested branches |

Each project exhibits the same seven problems on purpose, so the five reports
can be read side by side:

1. A function far over every complexity threshold
2. Deep control-flow nesting
3. A long parameter list
4. A long function
5. A block copy-pasted into a second file
6. A dependency cycle between two modules
7. A layering violation: the domain layer reaching into the persistence layer

## Results

Each project carries its own `.lumioguard-cc.json` with blocking thresholds,
because the shipped defaults are advisory and would only warn.

| Project | Files before → after | Source lines before → after | Source tokens before → after | Blocking findings before → after |
| --- | --- | --- | --- | --- |
| typescript-order-service | 6 → 10 | 258 → 307 | 1660 → 2143 | **10 → 0** |
| python-inventory | 9 → 10 | 154 → 185 | 1314 → 1549 | **9 → 0** |
| java-billing | 6 → 12 | 229 → 255 | 1480 → 1981 | **9 → 0** (1 advisory remains) |
| c-sensor-pipeline | 8 → 14 | 170 → 203 | 978 → 1156 | **9 → 0** |
| cpp-shipping-quotes | 8 → 12 | 147 → 196 | 898 → 1196 | **9 → 0** |

All 46 blocking findings are gone. The refactors cost between 11% and 33% more
source lines and between 18% and 34% more tokens, spread over roughly twice as
many functions: named types, records and lookup tables take room that nested
branches did not. The one advisory finding that remains in the Java example is a real
trade-off, discussed in its report.

## The examples are real code, not fixtures

Every version was checked with its language's own toolchain, so the measurements
describe code that actually compiles:

| Project | Check | before | after |
| --- | --- | --- | --- |
| typescript-order-service | `tsc --noEmit --strict` (TypeScript 5.9.2) | clean | clean |
| python-inventory | `python -m compileall` (CPython 3.13) | clean | clean |
| java-billing | `javac` (JDK 17.0.12) | 6 classes | 14 classes |
| c-sensor-pipeline | `zig cc -std=c11 -Wall -Wextra -Werror` (clang 21.1.0) | clean | clean |
| cpp-shipping-quotes | `zig c++ -std=c++17 -Wall -Wextra -Werror` (clang 21.1.0) | clean | clean |

The C and C++ refactors were also run against their bad versions with the same
inputs: 144 sensor batches and 6,912 quotes gave byte-identical output, so the
lower numbers describe the same behaviour.

One result is worth singling out. The Python `before/` version **cannot be
imported at all**:

```
ImportError: cannot import name 'StockLevel' from partially initialized module
'inventory.domain.stock' (most likely due to a circular import)
```

Every file compiles on its own, so a per-file linter sees nothing. The
`dependency.cycle` finding that lumioguard CC reports statically is the same
defect CPython hits at import time. After the refactor the module imports and
runs, moving 288 units in two transfers.

## Running the demo

From the repository root, after `make build` (or `go build -o bin/lumioguard-cc ./cmd/lumioguard-cc`):

```bash
# 1. The bad version fails the gate
lumioguard-cc check --root .examples/typescript-order-service/before        # exit 1

# 2. The refactored version passes on its own
lumioguard-cc check --root .examples/typescript-order-service/after         # exit 0
```

`make examples` runs all five projects, and adds a third step that measures the
refactor against the bad version. On hosts without make, `sh .examples/run.sh`
does the same and also asserts the exit codes.

## How step 3 compares two directories

These examples are laid out as `before/` and `after/` because both versions have
to sit in the repository side by side, and two directories share no history to
compare through. So the runner builds the history it needs: it copies `before/`
into a temporary directory, commits it, replaces the sources with `after/`, and
measures the difference with `--base HEAD`.

```bash
work=$(mktemp -d)
cp -R .examples/java-billing/before/. "$work/"
git -C "$work" init -q && git -C "$work" add -A
git -C "$work" commit -qm "the deliberately bad version"
find "$work" -mindepth 1 -maxdepth 1 ! -name .git ! -name .lumioguard-cc.json -exec rm -rf {} +
cp -R .examples/java-billing/after/. "$work/"
lumioguard-cc check --base HEAD --root "$work"
```

```
lumioguard CC check: PASSED
12 files analyzed in 839 ms
0 blocking, 1 advisory findings
Source tokens: 1981 (+501 since the base commit)
- [warning] .../BillingService.java: coupling.module_fan_out is 11 count; the configured threshold is 8 (new)
Resolved since the base commit: 13
```

Nothing is stored and nothing is left behind. The temporary directory is deleted
after the check, and the only file either version carries is
`.lumioguard-cc.json`, which is policy rather than measurements.

This is the same workflow the README recommends for real repositories, where you
refactor in place and Git already holds the previous version. A stored baseline
is worth creating only when a team wants a debt ceiling frozen at a reviewed
point rather than one that moves with every commit.

## This repository is the Go example

lumioguard CC is written in Go and checks itself. The root `.lumioguard-cc.json`
includes every `.go` file except the copied TypeScript parser and the generated
Java parser, declares the dependency rules from the development guide as
boundaries, and blocks every rule. `bin/lumioguard-cc check` on 2026-09-17:

| Measurement | Value |
| --- | --- |
| Files analyzed | 133 |
| Source tokens | 110856 |
| Blocking findings | 68 |
| Duplicated blocks (`duplication.token_clone`) | 0 reported: clone density is 4.9 %, under the 5 % threshold; the measurement's evidence counts 29 copied blocks |
| Functions over a complexity or size threshold | 43 |
| Files over 4000 tokens | 2, both in the hand-written Python parser |
| Boundary violations and cycles | 0 |

Those findings are the tool's own debt. Because the Claude Code hook and the
Build workflow compare with the previous commit, a change to this repository
fails only when it adds to that list, and reducing it is ordinary cleanup work.
Imports of the excluded parsers show up as `dependency.import_unresolved`
warnings, which is the documented behaviour for imports into excluded code.
