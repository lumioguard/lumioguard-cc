# Vendored TypeScript parser

A copy of the parser, scanner and AST packages of Microsoft TypeScript for Go (parser, scanner and AST packages) at version `v0.0.0-20260820064610-89d5d5b2849a`.

Do not edit these files by hand. Refresh them with:

    go run ./tools/tsgo-sync

Upstream keeps these packages under `internal/`, which Go does not allow other
modules to import. Copying them is permitted by the Apache License 2.0; the
LICENSE and NOTICE.txt files are reproduced here and every copied file carries
a modification notice.

## Modifications

- Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
- The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
- Test files omitted.

The pinned version is recorded in `internal/thirdparty/components.go` and
checked against the Go vulnerability database by `go run ./tools/third-party-audit`.
