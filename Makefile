GO      ?= go
BINARY  := lumioguard-cc
VERSION ?=
LDFLAGS := $(if $(VERSION),-ldflags "-X github.com/lumioguard/lumioguard-cc/internal/product.Version=$(VERSION)")
PKGS    := $(shell $(GO) list ./... | grep -v /internal/thirdparty/ | grep -v /internal/adapter/java/syntax)
DIRS    := $(shell $(GO) list -f '{{.Dir}}' ./... | grep -v thirdparty | grep -v 'java[/\\]syntax')

ifeq ($(OS),Windows_NT)
EXT := .exe
else
EXT :=
endif

.PHONY: build test vet lint fmt tidy sync-parser sync-java-grammar corpus-python corpus-java corpus-go docs docs-serve examples audit notices dist clean

build: ## Build the CLI into bin/; VERSION=x.y.z stamps a release version
	$(GO) build -trimpath $(LDFLAGS) -o bin/$(BINARY)$(EXT) ./cmd/lumioguard-cc

test: ## Run all tests (vendored and generated parsers excluded)
	$(GO) test $(PKGS)

vet: ## Static checks; the generated Java parser trips only the unreachable check
	$(GO) vet -unreachable=false ./...

lint: ## golangci-lint (must be installed separately)
	golangci-lint run ./...

fmt: ## Format own sources
	gofmt -l -w $(DIRS)

tidy:
	$(GO) mod tidy

sync-parser: ## Refresh the vendored TypeScript parser (pinned version in tools/tsgo-sync)
	$(GO) run ./tools/tsgo-sync

sync-java-grammar: ## Regenerate the Java parser (needs a Java runtime; pinned in tools/java-grammar-sync)
	$(GO) run ./tools/java-grammar-sync

corpus-python: ## Parse a Python corpus and compare with CPython: make corpus-python DIR=/path/to/Lib PYTHON=python3
	$(GO) run ./tools/parse-corpus -lang python -dir "$(DIR)" -python "$(PYTHON)"

corpus-java: ## Parse a Java corpus: make corpus-java DIR=/path/to/sources
	$(GO) run ./tools/parse-corpus -lang java -dir "$(DIR)"

corpus-go: ## Parse a Go corpus: make corpus-go DIR=/path/to/sources
	$(GO) run ./tools/parse-corpus -lang go -dir "$(DIR)"

audit: ## Security gate: linked modules plus the upstreams copied into this repository
	govulncheck ./...
	$(GO) run ./tools/third-party-audit

notices: ## Regenerate THIRD-PARTY-NOTICES.md, which must ship with the binary
	$(GO) run ./tools/third-party-notices

dist: build notices ## Build the binary next to the notices a distribution must carry
	@mkdir -p dist
	@cp bin/$(BINARY)$(EXT) dist/
	@cp THIRD-PARTY-NOTICES.md dist/
	@echo "dist/ contains the executable and the third-party notices it must ship with"

docs: ## Build the documentation site into dist/site (needs: pip install -r requirements-docs.txt)
	rm -rf dist/docs-src && mkdir -p dist && cp -R .documentations dist/docs-src
	zensical build --clean --strict
	test -f dist/site/rules/index.html

docs-serve: ## Preview the documentation at http://localhost:8000
	rm -rf dist/docs-src && mkdir -p dist && cp -R .documentations dist/docs-src
	zensical serve

examples: build ## Run every worked example: the bad version fails, the refactor passes
	@sh .examples/run.sh bin/$(BINARY)

clean: ## Remove build output
	rm -rf bin dist
