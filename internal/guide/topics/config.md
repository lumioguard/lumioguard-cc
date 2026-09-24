# Configuring .lumioguard-cc.json

`lumioguard-cc init` writes this file with the defaults below. Validation is strict: an unknown or
misspelled field stops the check with exit code 2 and names the field. Run a check after every
edit.

```json
{
  "schemaVersion": "1.0",
  "source": {
    "include": ["**/*.js", "**/*.jsx", "**/*.mjs", "**/*.cjs", "**/*.ts", "**/*.tsx", "**/*.mts", "**/*.cts",
                "**/*.py", "**/*.pyw", "**/*.java", "**/*.go", "**/*.c", "**/*.h", "**/*.cc",
                "**/*.cpp", "**/*.cxx", "**/*.c++", "**/*.hh", "**/*.hpp", "**/*.hxx", "**/*.h++"],
    "exclude": ["**/node_modules/**", "**/dist/**", "**/build/**", "**/out/**", "**/coverage/**",
                "**/.git/**", "**/.lumioguard-cc/**", "**/*.min.js", "**/*.generated.*",
                "**/.next/**", "**/.nuxt/**", "**/.output/**", "**/.svelte-kit/**", "**/.turbo/**",
                "**/.cache/**", "**/.parcel-cache/**", "**/__pycache__/**", "**/.venv/**",
                "**/venv/**", "**/site-packages/**", "**/target/**", "**/.gradle/**",
                "**/vendor/**", "**/testdata/**"]
  },
  "metrics": {
    "cyclomatic":         { "enabled": true, "threshold": 12,   "severity": "warning", "block": false },
    "cognitive":          { "enabled": true, "threshold": 15,   "severity": "warning", "block": false },
    "nestingDepth":       { "enabled": true, "threshold": 4,    "severity": "warning", "block": false },
    "functionLines":      { "enabled": true, "threshold": 80,   "severity": "warning", "block": false },
    "parameterCount":     { "enabled": true, "threshold": 5,    "severity": "warning", "block": false },
    "fileTokens":         { "enabled": true, "threshold": 4000, "severity": "warning", "block": false },
    "duplicationPercent": { "enabled": true, "threshold": 5,    "severity": "warning", "block": false,
                            "minTokens": 50, "minLines": 5 },
    "fanOut":             { "enabled": true, "threshold": 20,   "severity": "warning", "block": false }
  },
  "architecture": { "boundaries": [], "blockCycles": true, "blockViolations": true },
  "coverage": { "lcovFile": null, "required": false }
}
```

- A finding is raised when a value is **above** its `threshold`.
- `fileTokens` limits how much code one file holds, counted in lexical tokens without comments.
  It is the same measure that `size.total_tokens` sums for the whole project.
- `severity` is only a label. With `block: true`, a new or worse finding fails the check.
- `enabled: false` stops findings, but the value is still measured.
- `blockCycles` and `blockViolations` control whether new dependency cycles and boundary violations
  fail the check.

## Excluding code

Exclude only code that nobody edits by hand:

- generated code, such as `**/generated/**` or `**/*.pb.*`;
- vendored copies, such as `**/vendor/**` or `**/third_party/**`;
- build output and bundles the defaults do not cover, such as `**/storybook-static/**` or
  `**/*.bundle.js`.

Do not exclude tests, migrations or scripts just because they score badly. The team should see those
numbers.

## Architecture boundaries

Each boundary names a layer, the files in it, and the layers it may import:

- The first boundary whose `include` matches a file owns that file.
- A layer must list itself in `mayDependOn` if its files import each other.
- Files outside every boundary are not checked.

```json
"boundaries": [
  { "name": "domain",      "include": ["src/domain/**"],      "mayDependOn": ["domain"] },
  { "name": "persistence", "include": ["src/persistence/**"], "mayDependOn": ["domain", "persistence"] },
  { "name": "api",         "include": ["src/api/**"],         "mayDependOn": ["domain", "persistence", "api"] }
]
```

For a frontend, the layers are often `shared` (`src/shared/**`), `features` (`src/features/**`) and
`app` (`src/app/**`): each may import the layers listed before it.

Propose boundaries only when the folders already show the layers, and confirm them with the user.
Existing violations appear as findings, but a comparison blocks only on new ones.

## Coverage

If the tests write an LCOV report (Jest, Vitest, c8, nyc, `coverage lcov` for Python, or a
JaCoCo-to-LCOV converter for Java):

```json
"coverage": { "lcovFile": "coverage/lcov.info", "required": false }
```

With `required: true`, a missing or out-of-date report makes the result incomplete. Use it only in
CI, straight after the tests run.
