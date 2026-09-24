---
description: Fan-out, fan-in, dependency cycles, and how lumioguard CC follows imports.
---

# Dependencies

These rules look at how files import each other.

<div class="lg-summary" markdown>

| Rule | ID | Default | Measures |
| --- | --- | --- | --- |
| Fan-out | `coupling.module_fan_out` | 20, does not block | Project files one file imports |
| Fan-in | `coupling.module_fan_in` | Reported only | Project files that import one file |
| Dependency cycles | `dependency.cycle` | Any cycle blocks | Files that import each other in a loop |

</div>

## Fan-out and fan-in

**Fan-out** is how many project files a file imports. A file with a high fan-out can break when any of
those files changes, and often mixes several jobs.

**Fan-in** is how many project files import a file. A high fan-in is fine for stable shared code, and
risky for code that changes often.

Imports of packages and the standard library are not counted.

**To lower fan-out,** split the file by responsibility, or move the code that ties everything together
up to where the application starts.

## Dependency cycles

Two or more files that reach each other through imports, such as `order.ts` importing `database.ts`,
which imports `order.ts`.

- Files in a cycle cannot be understood, tested or reused on their own.
- In Python, a cycle can stop the program from starting.
- In Go, the compiler rejects import cycles between packages, so this rule only reports what the
  compiler would refuse anyway.
- In C and C++, two headers that include each other form a cycle even when include guards stop the
  compiler from looping.
- Every group of files that reach each other is reported once. A file that imports itself is not
  reported.
- Cycles block by default, through `architecture.blockCycles`.

### How to fix a cycle

- **Shared types:** move the types both files need into a third file that neither imports back.
- **A lower layer calling a higher one:** define an interface or callback in the lower layer, implement
  it in the higher one, and pass it in. The dependency then points one way.
- **In Python,** import the modules after the fix to confirm the program still starts.

## How imports are found

Each import is sorted into one of three kinds:

- **Internal:** it points at another analyzed file. It becomes an arrow in the dependency graph.
- **External:** it points at a package, the standard library or a path alias. It is only counted.
- **Unresolved:** it looks internal, but no file matches. It is reported as a warning.

| Language | Internal | External | Unresolved |
| --- | --- | --- | --- |
| JavaScript and TypeScript | Relative paths, trying `.ts .tsx .js .jsx .mjs .cjs` and `index` files, and mapping `.js` to `.ts` | Package names and `tsconfig` path aliases | A relative path with no matching file |
| Python | Relative imports; absolute imports of packages in the analyzed code, including `src` layouts; literal `importlib.import_module` calls | Packages that are not analyzed | A relative import with no match, or a missing module in an analyzed package |
| Java | Classes from package declarations, single and wildcard imports, and type names used from the same package | Wildcard imports themselves, and packages that are not analyzed | A single-class import into an analyzed package where the class does not exist |
| C and C++ | `#include "x"` next to the including file; then, as through an include path, `x` from the project root, or the one analyzed file whose path ends in `x` | `#include <x>` where `x` has no directory, such as `<stdio.h>`, and any include that matches no analyzed file | An include that matches more than one analyzed file, such as `"config.h"` in two folders |
| Go | Import paths under a module declared by a `go.mod` inside the project. A package import becomes a dependency on the package's first non-test file in name order, so fan-out counts packages | The standard library and modules that are not analyzed | An import path inside an analyzed module that matches no analyzed package |

Computed import paths and includes such as `#include CONFIG_HEADER`, `sys.path` changes, monorepo workspaces, `go.work` files, and Java types used
only through `var` or reflection are not followed.
