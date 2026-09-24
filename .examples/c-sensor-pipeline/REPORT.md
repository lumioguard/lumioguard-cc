# C: sensor pipeline

A batch processor for sensor readings where calibration, range checks, storage
and alerting were all branches inside one function, and the reading type reached
back into storage.

## Before: 9 blocking findings, exit code 1

```
lumioguard CC check: FAILED
8 files analyzed in 161 ms
9 blocking, 0 advisory findings
Source tokens: 978
- [error] src/domain/reading.h:4: domain is not allowed to depend on persistence (new)
- [error] src/service/pipeline.c:5: complexity.cognitive is 52 count; the configured threshold is 15 (new)
- [error] src/service/pipeline.c:5: complexity.cyclomatic is 22 count; the configured threshold is 10 (new)
- [error] src/service/pipeline.c:5: complexity.nesting_depth is 6 count; the configured threshold is 4 (new)
- [error] src/domain/reading.h|src/persistence/store.h: A dependency cycle connects src/domain/reading.h, src/persistence/store.h (new)
- [error] clone:src/service/pipeline.c::process_batch|src/service/report.c::report_print: The same 74-token block appears in 2 locations, 24 lines in all (new)
- [error] repository: duplication.token_clone_density is 14.12 percent; the configured threshold is 3 (new)
- [error] src/service/pipeline.c:5: size.function_lines is 73 lines; the configured threshold is 60 (new)
- [error] src/service/pipeline.c:5: size.parameter_count is 9 count; the configured threshold is 5 (new)
```

| Finding | Count |
| --- | --- |
| `duplication.token_clone` | 1 |
| `complexity.cyclomatic` | 1 |
| `complexity.cognitive` | 1 |
| `complexity.nesting_depth` | 1 |
| `size.function_lines` | 1 |
| `size.parameter_count` | 1 |
| `duplication.token_clone_density` | 1 |
| `dependency.cycle` | 1 |
| `architecture.boundary_violation` | 1 |

## After: 0 blocking findings, exit code 0

| Measurement | `process_batch` before | `process_batch` after |
| --- | --- | --- |
| Cyclomatic complexity | 22 | 3 |
| Cognitive complexity | 52 | 3 |
| Nesting depth | 6 | 2 |
| Source lines | 73 | 11 |
| Parameters | 9 | 3 |

| Repository measurement | Before | After |
| --- | --- | --- |
| Files | 8 | 14 |
| Nonblank source lines | 170 | 203 |
| Source tokens | 978 | 1156 |
| Measured functions | 6 | 14 |
| Clone density | 14.12% (24 lines, 1 group) | 0% |
| Worst cyclomatic complexity | 22 | 6 (`process_reading`) |
| Worst cognitive complexity | 52 | 5 (`process_reading`) |
| Worst nesting depth | 6 | 2 |
| Longest function | 73 lines | 17 lines (`process_reading`) |

Compared with the bad version through Git, the refactor resolves all 9 findings
and adds 178 source tokens.

## What changed

**Calibration moved into the domain.** The `switch` on the sensor kind, with an
`if` chain inside each case, became `calibrate_reading` in `calibration.c`,
which dispatches to one small function per kind. A humidity value outside 0 to
100 is still rejected, now by a return value instead of a `continue` four levels
down.

**The range rules became `struct limits`.** `limits_contains` and
`limits_is_critical` name the two questions the loop kept asking, including the
"far outside the band" rule that decides whether an alert is critical.

**The nine arguments became `struct batch_request`.** The sensor kind, the band,
the calibration flag and the alert mode travel together, and the magic alert
numbers 1 and 2 became `enum alert_mode`. `process_batch` takes the request, the
store and the result.

**One reading at a time.** `process_reading` handles a single reading with early
returns, and `record_rejection` owns the alert decision. The loop in
`process_batch` only filters by kind.

**The summary was printed twice.** `process_batch` ended with a verbatim copy of
`report_print`. It now calls `report_print`, and the separator lines are
constants. The mean is computed once, by `batch_mean`.

**The cycle was a convenience function.** `reading_reload(struct reading *,
struct store *)` made `reading.h` include the store's header, which includes
`reading.h` back. Include guards let that compile, but it tied the domain to
storage. Deleting `reading_reload` removed the cycle and the boundary
violation; callers that need to reload already hold the store.

## Behaviour did not change

Both versions were built with clang 21.1.0 (`zig cc -std=c11 -Wall -Wextra
-Werror`) and run with the same driver: 500 generated readings, every sensor
kind, three bands, with and without calibration, all three alert modes, with and
without a store. That is 144 batches, and standard output and standard error
were byte for byte identical.
