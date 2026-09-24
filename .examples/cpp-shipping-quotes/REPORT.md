# C++: shipping quotes

A quote service where every carrier, zone and weight band was a branch inside
one method, and the parcel type reached back into the quote cache.

## Before: 9 blocking findings, exit code 1

```
lumioguard CC check: FAILED
8 files analyzed in 212 ms
9 blocking, 0 advisory findings
Source tokens: 898
- [error] src/domain/Parcel.hpp:5: domain is not allowed to depend on persistence (new)
- [error] src/service/QuoteService.cpp:5: complexity.cognitive is 40 count; the configured threshold is 15 (new)
- [error] src/service/QuoteService.cpp:5: complexity.cyclomatic is 22 count; the configured threshold is 10 (new)
- [error] src/service/QuoteService.cpp:5: complexity.nesting_depth is 6 count; the configured threshold is 4 (new)
- [error] src/domain/Parcel.hpp|src/persistence/QuoteCache.hpp: A dependency cycle connects src/domain/Parcel.hpp, src/persistence/QuoteCache.hpp (new)
- [error] clone:src/service/QuoteService.cpp::QuoteService.quote|src/service/Receipt.cpp::Receipt.print: The same 60-token block appears in 2 locations, 20 lines in all (new)
- [error] repository: duplication.token_clone_density is 13.61 percent; the configured threshold is 3 (new)
- [error] src/service/QuoteService.cpp:5: size.function_lines is 68 lines; the configured threshold is 60 (new)
- [error] src/service/QuoteService.cpp:5: size.parameter_count is 7 count; the configured threshold is 5 (new)
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

| Measurement | `QuoteService::quote` before | `QuoteService::quote` after |
| --- | --- | --- |
| Cyclomatic complexity | 22 | 3 |
| Cognitive complexity | 40 | 2 |
| Nesting depth | 6 | 1 |
| Source lines | 68 | 13 |
| Parameters | 7 | 1 |

| Repository measurement | Before | After |
| --- | --- | --- |
| Files | 8 | 12 |
| Nonblank source lines | 147 | 196 |
| Source tokens | 898 | 1196 |
| Measured functions | 8 | 18 |
| Clone density | 13.61% (20 lines, 1 group) | 0% |
| Worst cyclomatic complexity | 22 | 4 (`postalPrice`, `expressMultiplier`) |
| Worst cognitive complexity | 40 | 3 |
| Worst nesting depth | 6 | 2 (`postalPrice`) |
| Longest function | 68 lines | 13 lines (`QuoteService::quote`) |

Compared with the bad version through Git, the refactor resolves all 9 findings
and adds 298 source tokens.

The insurance lambda inside the old `quote` was measured as its own function,
`QuoteService.quote.insurance`, and its branches also raised the method's
cognitive complexity, as nested functions do in every language.

## What changed

**Carriers became a table of pricing functions.** The `if (carrier == ...)`
chain is now a `std::map` from carrier name to a pricing function in
`Carriers.cpp`. An unknown carrier is a failed lookup instead of a final `else`.
The courier's zones are a `switch` on `enum class Zone`, and the long-haul case,
which held the deepest nesting, is its own function.

**Surcharges became three named rules.** `expressMultiplier`, `insurance` and
`fragileFee` in `Surcharges.cpp` replace the flags checked at the end of
`quote`, and the weekend test is `QuoteRequest::isWeekend`.

**The seven arguments became `QuoteRequest`.** The carrier, parcel, zone, express
and insurance flags and the weekday travel together. `quote` returns
`std::optional<double>` instead of the sentinel `-1`.

**The receipt was printed twice.** `quote` ended with a copy of
`Receipt::print`. `QuoteService` now holds a `Receipt` and calls it, and the
separator lines are constants.

**The cycle was a convenience method.** `Parcel::lastQuote(const QuoteCache&)`
made `Parcel.hpp` include the cache's header, which includes `Parcel.hpp` back.
`#pragma once` and a forward declaration let it compile, but it tied the
domain to storage. The service asks the cache directly now, so the domain
headers include nothing from persistence.

## Behaviour did not change

Both versions were built with clang 21.1.0 (`zig c++ -std=c++17 -Wall -Wextra
-Werror`) and run with the same driver: four carriers (one unknown), three
zones, eight weights, three declared values, express, insurance, fragile and
three weekdays, 6,912 quotes in all. The prices, the 1,800 declined quotes and
the 5,112 printed receipts were byte for byte identical.
