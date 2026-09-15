---
description: Function length and parameter count, with a worked example.
---

# Size

Two rules flag functions that do too much: too many lines, or too many inputs. A third flags files
that hold too much code to read in one go, and a fourth reports how much code the project has in all.

<div class="lg-summary" markdown>

| Rule | ID | Default | Measures |
| --- | --- | --- | --- |
| Function length | `size.function_lines` | 80 | Lines of code in a function |
| Parameter count | `size.parameter_count` | 5 | Parameters a function takes |
| File tokens | `size.file_tokens` | 4000 | Source tokens in a file |
| Total tokens | `size.total_tokens` | Reported only | Source tokens in the whole project |

</div>

## Worked example

```ts
// A comment line is not counted.
export function label(name: string, city: string, zip: string, country: string) {

  return `${name}, ${city} ${zip}, ${country}`;
}
```

The tool reports a **function length of 3** and a **parameter count of 4** for this code.

## Function length

Lines of code from the first line of a function to its last. Blank lines and comment-only lines are not
counted.

- The count starts at the first word of the declaration, which includes modifiers and annotations in
  TypeScript and Java. In Python it starts at `def`, so decorators are not counted.
- A nested function's lines count in the outer function as well as on their own.

Length is a hint, not a verdict: a long table of data is long without being complicated.

## Parameter count

Each declared parameter counts once, whatever its form: default values, rest and destructured
parameters, Python `*args`, `**kwargs`, keyword-only parameters, `self` and `cls`, and TypeScript's
explicit `this`. In Go, `(a, b int)` counts two and a variadic parameter counts one. Java and Go
receivers do not count.

Passing one object with ten fields counts as 1.

## File tokens

The number of source tokens in a file: identifiers, keywords, literals and punctuation, with comments
and whitespace removed. It is the same token stream the [duplication](duplication.md) rule reads.

This is the closest the tool gets to what a coding agent has to take in before it can change a file.
Lexical tokens are not a language model's tokens, but the two rise and fall together, so a file that
grows from 3,000 to 6,000 tokens costs an agent about twice as much to read on every later task.

`size.total_tokens` adds the counts of every analyzed file. It has no threshold, but the check summary
prints it, and a comparison shows the difference:

```text
Source tokens: 12480 (+312 since the base commit)
```

If any file could not be analyzed, neither number is reported, so an incomplete run never looks
smaller than a complete one.

## How to lower it

**Long functions:**

- Extract each step into a function whose name says what it does.
- Move long literal tables, such as lists of rates, out of the function.
- Do not split in the middle of a step just to get under the limit. Two confusing halves are worse
  than one clear function.

**Long parameter lists:**

- Group parameters that always travel together into one object, such as an `address` or `options`.
- If a parameter is only passed through to another call, pass the object that owns it instead.
- For a public function that other code depends on, keep the old signature as a thin wrapper while
  callers move over.

**Large files:**

- Split the file by responsibility, so each new file has one reason to change.
- Move large literal tables, such as fixtures or lookup data, into their own file or a data file.
- Do not spread one function over several files to get under the limit. A file is too big when it
  holds too many jobs, not too many lines of one job.
