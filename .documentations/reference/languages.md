---
description: Supported languages and file types, and the limits of what a check can tell you.
---

# Languages and limits

## Supported languages

| Language | File types | Parser |
| --- | --- | --- |
| JavaScript and TypeScript | `.js .jsx .mjs .cjs .ts .tsx .mts .cts` | Microsoft's TypeScript parser, from the typescript-go project |
| Python 3 | `.py .pyw` | Written for this tool, tested against the CPython 3.13 standard library |
| Java up to 17 | `.java` | Generated from the community ANTLR Java grammar |
| Go | `.go` | The Go standard library's `go/parser`, as built into the release |

All four parsers are compiled into the one program. You do not need Node.js, Python, Java or Go
installed. A project that mixes languages is checked in one run.

## Limits by language

| Language | Limits |
| --- | --- |
| JavaScript and TypeScript | JSX only in `.jsx` and `.tsx` files. Package `exports`, workspaces and path aliases count as external. Computed imports are not followed. |
| Python | Python 3 only. Code inside f-string fields is not analyzed. `sys.path` changes, namespace packages and installed packages are not modelled. `.pyi` files are skipped. |
| Java | Java 17 grammar, so newer syntax may fail to parse. Types seen only through `var` or reflection are missed. Initializer blocks, implicit record constructors and method references are not measured. The Java parser is slower than the others. |
| Go | Imports are resolved through `go.mod` files inside the analyzed folder; without one, every import is external. `go.work` files, build tags and cgo are not modelled. An import of a package becomes a dependency on the package's first non-test file in name order. Assembly-backed functions without a body are not measured. |

## What a check does not prove

A passing check does **not** mean the code:

- is correct, secure or well tested;
- is well designed, or has no technical debt;
- has good documentation, performance or up-to-date dependencies.

Other things to keep in mind:

- **Functions are identified by file and name.** Renaming a function, or moving and editing its file in
  one change, makes its findings look new.
- **Only exact copies and traceable imports are seen.** Renamed copies and computed imports are missed.
- **Coverage shows code that ran, not code that was tested well.** The staleness check compares file
  times only.
- **Settings can be changed.** Anyone with write access, including a coding agent, can edit the
  configuration or a stored baseline. Real enforcement comes from reviewing those files and running the
  same check in CI.
- **Very large repositories:** `--base` reads the earlier commit into memory, which can be slow.
- **Platforms:** the tool has been validated on Windows. macOS and Linux builds are published but not
  yet tested.

## What the tool never does

- Run your code or your tests.
- Connect to the network or call an AI model.
- Change source files, configuration or Git state during a check. Only `init` and `baseline create`
  write files, and neither overwrites anything unless told to.
