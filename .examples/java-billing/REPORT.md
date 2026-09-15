# Java: billing

An invoice calculator where every tax rule, coupon and shipping rate was a
branch inside one method, and the invoice type reached back into storage.

## Before: 13 blocking findings, exit code 1

```
lumioguard CC check: FAILED
6 files analyzed in 341 ms
13 blocking, 0 advisory findings
Source tokens: 1480
- [error] .../domain/Invoice.java:3: domain is not allowed to depend on persistence (new)
- [error] .../service/BillingService.java:14: complexity.cognitive is 77 count; the configured threshold is 15 (new)
- [error] .../service/BillingService.java:14: complexity.cyclomatic is 32 count; the configured threshold is 10 (new)
- [error] .../service/BillingService.java:14: complexity.nesting_depth is 7 count; the configured threshold is 4 (new)
- [error] .../domain/Invoice.java|.../persistence/InvoiceStore.java: A dependency cycle connects ... (new)
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

## After: 0 blocking findings, exit code 0

| Measurement | `calculateInvoice` before | `calculateInvoice` after |
| --- | --- | --- |
| Cyclomatic complexity | 32 | 1 |
| Cognitive complexity | 77 | 0 |
| Nesting depth | 7 | 0 |
| Source lines | 102 | 15 |
| Parameters | 8 | 1 |

| Repository measurement | Before | After |
| --- | --- | --- |
| Files | 6 | 12 |
| Nonblank source lines | 229 | 255 |
| Source tokens | 1480 | 1981 |
| Measured functions | 13 | 32 |
| Clone density | 18.3% (42 lines, 5 groups) | 0% |
| Worst cyclomatic complexity | 32 | 5 (`problemFor`, `shippingFor`) |
| Worst cognitive complexity | 77 | 4 |
| Worst nesting depth | 7 | 2 (`priceItems`) |
| Longest function | 102 lines | 16 lines (`priceItems`) |

`calculateInvoice` now has no branches at all: it prices the items, asks three
policies for numbers, builds the invoice and saves it.

## What changed

**Tax rules became four maps.** `standardRate` and `reducedRate` were chains of
`if (region.equals(...))`; they are now `Map.getOrDefault` lookups against
`US_STANDARD_BY_REGION`, `US_REDUCED_BY_REGION`, `STANDARD_BY_COUNTRY` and
`REDUCED_BY_COUNTRY`. `TaxRules.standardRate` fell from cyclomatic 5 to 2 and
its nesting from 4 to 1. The country-and-region pair travels as `TaxContext`,
which also answers `isUnitedStates()` so the string comparison appears once.

**The coupon switch became a rule table.** Four `case` labels with nested
conditions are now a `Map<String, Rule>` of small lambdas in `DiscountPolicy`.
Each lambda is measured separately and none exceeds cyclomatic 2.

**Shipping became a rate record per country.** `ShippingPolicy.shippingFor`
takes a `ShippingRequest` and returns early for waived shipping and express
orders, so the free-shipping threshold is evaluated in one place.

**The eight-argument call became `OrderRequest`.** Java records make this cheap,
and `OrderRequest` is also where `isTaxExempt()` and `taxContext()` live, so the
service no longer assembles them.

**`summarise` was a verbatim copy of `Reporting.render`.** Both built the same
separator lines, totals and optional detail list. `BillingService.summarise` now
delegates in one line, and `Reporting` keeps the separators as constants. Those
were 42 of 229 source lines, hence 18.3% clone density.

**The cycle was a convenience method.** `Invoice.reload(InvoiceStore)` made the
domain record import the persistence layer, and `InvoiceStore` imports `Invoice`
because it stores them. Deleting `reload` removed the cycle and the layering
violation; callers that need to reload already hold the store.

## The advisory finding that remains

```
- [warning] .../service/BillingService.java: coupling.module_fan_out is 11 count; the configured threshold is 8
```

This is real and was accepted rather than hidden. Splitting a god method into
named collaborators means the orchestrator imports all of them: `BillingService`
now depends on nine domain types, the store and the reporter. The obvious way to
make the number go down is a facade that re-exports the domain types, which
would add a layer of indirection without reducing the actual coupling, and would
make the metric lie.

The threshold is configured as `"severity": "warning", "block": false`, so the
finding is visible in every report and does not fail the build. That is the
distinction the tool is built around: a heuristic that deserves a look is not the
same as a rule the project has decided to enforce. If this codebase grew, the
right response would be to split `BillingService` by use case, not to raise the
threshold quietly.
