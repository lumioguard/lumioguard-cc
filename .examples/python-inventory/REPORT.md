# Python: warehouse inventory

A stock rebalancing routine that decides which warehouses have too much of a
product, which have too little, and moves units between them. The whole
decision lived in one function with the report generator inlined at the end.

## Before: 13 blocking findings, exit code 1

```
lumioguard CC check: FAILED
9 files analyzed in 164 ms
13 blocking, 0 advisory findings
Source tokens: 1314
- [error] inventory/domain/stock.py:3: domain is not allowed to depend on persistence (new)
- [error] inventory/service/manager.py:6: complexity.cognitive is 75 count; the configured threshold is 15 (new)
- [error] inventory/service/manager.py:6: complexity.cyclomatic is 25 count; the configured threshold is 10 (new)
- [error] inventory/service/manager.py:6: complexity.nesting_depth is 7 count; the configured threshold is 4 (new)
- [error] inventory/domain/stock.py|inventory/persistence/store.py: A dependency cycle connects inventory/domain/stock.py, inventory/persistence/store.py (new)
```

| Finding | Count |
| --- | --- |
| `duplication.token_clone` | 5 |
| `complexity.cyclomatic` | 1 |
| `complexity.cognitive` | 1 |
| `complexity.nesting_depth` | 1 |
| `size.function_lines` | 1 |
| `size.parameter_count` | 1 |
| `duplication.token_clone_density` | 1 |
| `dependency.cycle` | 1 |
| `architecture.boundary_violation` | 1 |

## After: 0 findings, exit code 0

| Measurement | `rebalance_warehouses` before | Worst function after |
| --- | --- | --- |
| Cyclomatic complexity | 25 | 7 (`_positions`) |
| Cognitive complexity | 75 | 8 (`_fill_deficit`) |
| Nesting depth | 7 | 3 (`_positions`) |
| Source lines | 84 | 21 (`_positions`) |
| Parameters | 8 | 4 (`write_transfer`) |

| Repository measurement | Before | After |
| --- | --- | --- |
| Files | 9 | 10 |
| Nonblank source lines | 154 | 185 |
| Source tokens | 1314 | 1549 |
| Measured functions | 13 | 17 |
| Clone density | 18.8% (29 lines, 5 groups) | 0% |

The entry point the rest of the service calls, `rebalance`, is now cyclomatic 2,
cognitive 1, 11 lines, 3 parameters.

## What changed

**The decision became a domain rule with a name.** The original asked, inline,
whether a warehouse was above its reorder point plus tolerance, whether the
product was prioritised, and whether it had demand. That is now
`rules.classify`, which returns a `Position` holding either a surplus, a deficit,
or a reason for sitting the round out. It is 11 lines and cyclomatic 6.

**The planning loop became three small functions.** `plan_transfers` walks the
products, `_plan_sku` handles one product, `_positions` splits its warehouses
into givers and takers, and `_fill_deficit` moves units. Nesting fell from 7 to
3 because each function owns one level of the problem.

**The eight-parameter signature became `PlanRequest`.** Levels, warehouses,
priority products, season, transfer cap and tolerance travel together with
defaults for the last two, so callers pass four values.

**The report generator was deleted from the planner.** `manager.py` built a
report with the same fifteen lines of `lines.append` calls that
`reporting.py` already had. `render_report` is now the only place a plan becomes
text, and it takes the `Plan` object rather than four parallel lists. Those
fifteen lines were 29 of 154 source lines, which is why clone density was 18.8%.

**The import cycle was a type-versus-data confusion.** `domain/stock.py`
imported `persistence/store.py` to read rows, and `persistence/store.py` imported
`domain/stock.py` for the `StockLevel` type. `domain/stock.py` now holds the
dataclass and nothing else; the persistence layer builds `StockLevel` values and the
service passes them into the domain. The cycle and the layering violation both
disappear.

This one was not a style preference. The `before/` package cannot be imported:

```
$ python -c "from inventory.service.manager import run"
ImportError: cannot import name 'StockLevel' from partially initialized module
'inventory.domain.stock' (most likely due to a circular import)
```

`python -m compileall` passes, because every file is valid on its own. A
per-file check cannot see this; a dependency graph can. After the refactor the
package imports and `run(["north", "south"], ["FRAGILE-2"], "summer")` moves 288
units in two transfers.

**`low_stock` stopped reaching for its own data.** It took a threshold and
called `load_levels()` itself; it now takes the levels. The function is
unchanged in complexity but is testable without the persistence layer.

## A note on Python-specific measurement

Comprehension clauses count towards cyclomatic complexity but not towards
cognitive complexity or nesting depth, which is why `_positions` scores 7 and 7
on the two complexity metrics despite containing a dictionary comprehension and
two sort keys. The exact rules are in `.documentations/rules/complexity.md`;
`lumioguard-cc explain complexity.cyclomatic` prints them on demand.
