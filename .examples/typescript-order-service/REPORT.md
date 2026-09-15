# TypeScript: order service

An order endpoint that grew one branch at a time until a single function did
validation, pricing, tax, discounts, shipping, persistence and email.

## Before: 16 blocking findings, exit code 1

```
lumioguard CC check: FAILED
6 files analyzed in 156 ms
16 blocking, 0 advisory findings
Source tokens: 1660
- [error] src/domain/order.ts:1: domain is not allowed to depend on persistence (new)
- [error] src/api/handler.ts:5: complexity.cognitive is 96 count; the configured threshold is 15 (new)
- [error] src/api/handler.ts:5: complexity.cyclomatic is 52 count; the configured threshold is 10 (new)
- [error] src/api/validation.ts:9: complexity.cyclomatic is 16 count; the configured threshold is 10 (new)
- [error] src/api/handler.ts:5: complexity.nesting_depth is 7 count; the configured threshold is 4 (new)
- [error] src/domain/order.ts|src/persistence/database.ts: A dependency cycle connects src/domain/order.ts, src/persistence/database.ts (new)
```

| Finding | Count |
| --- | --- |
| `duplication.token_clone` | 7 |
| `complexity.cyclomatic` | 2 |
| `complexity.cognitive` | 1 |
| `complexity.nesting_depth` | 1 |
| `size.function_lines` | 1 |
| `size.parameter_count` | 1 |
| `duplication.token_clone_density` | 1 |
| `dependency.cycle` | 1 |
| `architecture.boundary_violation` | 1 |

## After: 0 findings, exit code 0

| Measurement | `processOrder` before | `processOrder` after |
| --- | --- | --- |
| Cyclomatic complexity | 52 | 5 |
| Cognitive complexity | 96 | 4 |
| Nesting depth | 7 | 1 |
| Source lines | 150 | 30 |
| Parameters | 10 | 2 |

| Repository measurement | Before | After |
| --- | --- | --- |
| Files | 6 | 10 |
| Nonblank source lines | 258 | 307 |
| Source tokens | 1660 | 2143 |
| Measured functions | 11 | 27 |
| Clone density | 15.5% (40 lines, 7 groups) | 0% |
| Worst cyclomatic complexity | 52 | 6 (`shippingFor`) |
| Worst cognitive complexity | 96 | 5 (`priceItems`) |
| Worst nesting depth | 7 | 2 (`postalCodeError`) |
| Longest function | 150 lines | 30 lines (`processOrder`) |
| Most parameters | 10 | 4 (`taxFor`) |

## What changed

**The ten-argument signature became a request object.** `processOrder(request,
deps)` takes what the order is and where its collaborators come from. The
address fields moved into `AddressInput`, which the validator already used.

**Tax, discount and shipping became lookup tables.** Three nested chains of
`if (country === "US") … else if (country === "CA") … else` became
`RATES_BY_COUNTRY[country] ?? DEFAULT_RATES` in `src/domain/tax.ts`,
`src/domain/discount.ts` and `src/domain/shipping.ts`. Adding a country is now a
data change. This is where most of the 52 cyclomatic points went.

**The item loop became two functions.** `itemProblem` answers "is this item
usable", `priceItems` accumulates totals and tax. Six levels of nesting became
a loop with two guard clauses and `continue`.

**The copy-pasted validation block was deleted.** The handler had an inline copy
of the address checks that `validation.ts` already performed; the handler now
calls `validateAddress`. That block was 40 of the 258 source lines, which is why
clone density was 15.5%.

**`validateAddress` itself was over the threshold** at cyclomatic 16, which is
easy to miss when a much worse function sits next to it. Its six-branch
`else if` chain became a required-field table and a postal-code pattern map, and
it dropped to 3.

**The cycle and the layering violation had one cause.** `domain/order.ts`
imported `persistence/database.ts` to load and save, and `persistence/database.ts` imported
`domain/order.ts` for its types. `src/domain/ports.ts` now declares
`CustomerRepository`, `OrderRepository` and `Mailer`; `persistence/` and
`notification/` implement them and the domain depends on nothing. The dependency points inwards, the cycle
is gone, and the boundary rule passes.

## The trade-off the numbers show

Fan-in on `src/domain/order.ts` rose from 2 to 5 and on `src/domain/ports.ts` it
is 3. More modules now depend on the domain types, which is the intended shape
for this design: a stable centre that many modules use, rather than two modules
importing each other. Fan-out stayed under the threshold everywhere.
