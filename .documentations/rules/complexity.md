---
description: Cyclomatic complexity, cognitive complexity and nesting depth, with a worked example.
---

# Complexity

Three rules measure how complicated a function's control flow is. They share one implementation for
every language, so a value of 7 means the same thing in TypeScript, Python, Java, Go, C and C++.

<div class="lg-summary" markdown>

| Rule | ID | Default | Measures |
| --- | --- | --- | --- |
| Cyclomatic complexity | `complexity.cyclomatic` | 12 | How many paths run through the function |
| Cognitive complexity | `complexity.cognitive` | 15 | How hard the function is for a person to follow |
| Nesting depth | `complexity.nesting_depth` | 4 | How deeply its conditions and loops are nested |

</div>

## Worked example

The numbers below are the tool's real output for this function.

```ts linenums="1"
interface Item { fragile: boolean; weight: number }
interface Order { items: Item[]; express: boolean }

export function shippingCost(order: Order): number {
  if (order.items.length === 0) {
    return 0;
  }
  let cost = 5;
  for (const item of order.items) {
    if (item.fragile && item.weight > 10) {
      cost += 8;
    } else if (item.fragile) {
      cost += 3;
    }
  }
  return order.express ? cost * 2 : cost;
}
```

**Cyclomatic complexity 7, cognitive complexity 7, nesting depth 3.**

## Cyclomatic complexity

The number of independent paths through a function. Each path is something to understand and to test.

**How it counts.** Start at 1, then add 1 for each:

- `if`, `else if` or `elif`;
- loop;
- `catch` or `except` handler;
- `case` other than `default` (in Go, `case 1, 2:` is one case);
- conditional expression (`a ? b : c`);
- `&&` or `||` (`and`, `or` in Python; also `??` in JavaScript and TypeScript);
- in Python, each `for` and `if` inside a comprehension.

**In the example:** 1, plus the `if` on line 5, the loop on line 9, the `if` and its `&&` on line 10,
the `else if` on line 12, and the conditional on line 16. Branches inside a nested function do not add
to the outer function.

!!! note
    Tools disagree about counting `&&` and `||`. Do not compare these numbers with another tool's.

## Cognitive complexity

A score for how hard the code is to read. It charges for every break in straight-line reading, and
charges more when that break is nested.

**How it counts:**

- **+1** for `if`, `else if`, `else`, a conditional expression, `switch` or `match`, each loop, each
  `catch` or `except`, and each `break` or `continue` to a label.
- **+1 for each level of nesting** on `if`, conditional expressions, `switch`, loops and `catch`.
  `else if` and `else` never get this extra charge.
- **+1** for each run of the same logical operator: `a && b && c` is one run; `a && b || c` is two.
- A nested function raises the nesting level for its contents, and its points count toward the outer
  function too.

**In the example:**

| Line | Construct | Points |
| --- | --- | --- |
| 5 | `if` | 1 |
| 9 | `for` loop | 1 |
| 10 | `if`, nested inside the loop | 1 + 1 |
| 10 | `&&` run | 1 |
| 12 | `else if` | 1 |
| 16 | conditional expression | 1 |
| | **Total** | **7** |

The JSON report lists every point with its line. This is an independent implementation of the
Cognitive Complexity specification published by SonarSource (version 1.5), with no SonarSource code. It
was checked against the specification's examples, not against SonarQube, so edge cases can differ.
Recursion, `??` and Python comprehensions are not counted.

## Nesting depth

The deepest level of nesting in a function. `if`, loops, `switch` or `match`, `catch` or `except`, and
conditional expressions each add a level, and `else if` counts one level deeper than its `if`. Nested
functions start again from zero, and Python comprehensions do not count.

**In the example:** the loop is level 1, the `if` level 2, and the `else if` level 3. Tools that treat
`else if` as flat report lower numbers.

## How to lower it

- **Return early.** Handle invalid or trivial cases first, instead of wrapping the main path in an `if`.
- **Replace chains of `if` or `switch` with a lookup table** when they map a value to a result.
- **Move a loop body into its own function**, so the loop reads as one line.
- **Name parts of long conditions:** `const eligible = isActive && hasBalance;`.
- **Split a function that does several jobs in a row**, such as validate, calculate, save and notify,
  and keep the original as a short coordinator.

Start with the lines that carry the deepest nesting. The JSON report's `evidence` shows them.

## Language notes

| Construct | JavaScript and TypeScript | Python | Java | Go | C and C++ |
| --- | --- | --- | --- | --- | --- |
| Else-if | `else if` | `elif` | `else if` | `else if` | `else if` |
| Conditional | `a ? b : c` | `b if a else c` | `a ? b : c` | none | `a ? b : c`, and GNU `a ?: b` |
| Switch | `switch`, `case`, `default` | `match`, `case`, `case _:` as default | `switch` statements and expressions; each `case` label counts | `switch`, type `switch` and `select`; each clause counts once, `default` never | `switch`; each `case` label counts |
| Loop | `for`, `for-in`, `for-of`, `while`, `do-while` | `for`, `async for`, `while`; a loop's `else` adds 1 cognitive | `for`, enhanced `for`, `while`, `do-while` | `for`, `for range` | `for`, range `for`, `while`, `do-while` |
| Catch | `catch` | each `except` or `except*` | each `catch`; a multi-catch is one | none | each `catch`, and MSVC `__except` |
| Labelled jump | `break label`, `continue label` | none | `break label`, `continue label` | `break label`, `continue label`, `goto` | `goto` |
| Logical operators | `&&`, `\|\|`, and `??` for cyclomatic only | `and`, `or` | `&&`, `\|\|` | `&&`, `\|\|` | `&&`, `\|\|`, and `and`, `or` in C++ files |

Python f-strings are read as one piece, so conditions inside `{...}` are not counted. In Go, a
function literal is measured on its own and also raises the nesting of the function that holds it, like
a nested function in any other language.

In C and C++, `if constexpr` counts as an `if`, and a lambda is measured on its own like a nested
function. `&&` in a reference declaration, such as `auto&& x` or `T&& x = f()`, is not a condition
and is not counted. A macro followed by a block and then `else` counts as an `if`, because only an
`if` takes an `else`; any other macro followed by a block counts nothing for the macro itself.
