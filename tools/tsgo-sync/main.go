// Command tsgo-sync copies Microsoft's typescript-go parser packages into internal/thirdparty/tsgo
// with rewritten imports and Apache-2.0 modification headers. Rerun after changing the pin.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lumiostack/lumioguard-cc/internal/paths"
	"github.com/lumiostack/lumioguard-cc/internal/thirdparty"
)

const (
	upstreamPrefix = "github.com/microsoft/typescript-go/internal/"
	localPrefix    = "github.com/lumiostack/lumioguard-cc/internal/thirdparty/tsgo/"
	packageName    = "tsgo"
)

// stdlibReplacements swap the experimental JSON module for encoding/json/v2, its standard
// library home. Longest path first, so a parent path cannot shadow its child.
var stdlibReplacements = [][2]string{
	{`"github.com/go-json-experiment/json/jsontext"`, `"encoding/json/jsontext"`},
	{`"github.com/go-json-experiment/json"`, `json "encoding/json/v2"`},
}

// packages is the in-module closure of internal/parser and internal/scanner at the pinned
// version, as listed by go list -deps on those two packages.
var packages = []string{
	"ast",
	"collections",
	"core",
	"debug",
	"diagnostics",
	"jsnum",
	"json",
	"locale",
	"parser",
	"scanner",
	"spanmap",
	"stringutil",
	"tspath",
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "tsgo-sync:", err)
		os.Exit(1)
	}
}

func run() error {
	component := thirdparty.TypeScriptParser
	root, err := thirdparty.RepositoryRoot()
	if err != nil {
		return err
	}
	source, err := moduleDirectory(component.Module, component.Version)
	if err != nil {
		return err
	}
	target := paths.Resolve(root, component.Path)
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("clean %s: %w", target, err)
	}
	for _, pkg := range packages {
		if err := copyPackage(component, filepath.Join(source, "internal", pkg), filepath.Join(target, pkg)); err != nil {
			return fmt.Errorf("copy package %s: %w", pkg, err)
		}
	}
	for _, name := range []string{"LICENSE", "NOTICE.txt"} {
		if err := copyFile(filepath.Join(source, name), filepath.Join(target, name), nil); err != nil {
			return err
		}
	}
	if err := writeReadme(component, target); err != nil {
		return err
	}
	fmt.Printf("synced %d packages from %s@%s into %s\n", len(packages), component.Module, component.Version, component.Path)
	return nil
}

func moduleDirectory(module, version string) (string, error) {
	if _, err := exec.Command("go", "mod", "download", module+"@"+version).Output(); err != nil {
		return "", fmt.Errorf("download %s@%s: %w", module, version, err)
	}
	output, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", module+"@"+version).Output()
	if err != nil {
		return "", fmt.Errorf("locate %s@%s in the module cache: %w", module, version, err)
	}
	dir := strings.TrimSpace(string(output))
	if dir == "" {
		return "", fmt.Errorf("module %s@%s is not in the module cache", module, version)
	}
	return dir, nil
}

func copyPackage(component thirdparty.Component, source, target string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			// Non-package subdirectories hold assets referenced by go:embed
			// directives (for example diagnostics/loc/*.json.gz).
			if name == "testdata" || isGoPackageDir(filepath.Join(source, name)) {
				continue
			}
			if err := copyAssets(filepath.Join(source, name), filepath.Join(target, name)); err != nil {
				return err
			}
			continue
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		rewrite := func(data []byte) []byte { return rewriteGoSource(component, data) }
		if err := copyFile(filepath.Join(source, name), filepath.Join(target, name), rewrite); err != nil {
			return err
		}
	}
	return nil
}

func isGoPackageDir(dir string) bool {
	matches, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	return len(matches) > 0
}

func copyAssets(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		return copyFile(path, filepath.Join(target, relative), nil)
	})
}

func copyFile(source, target string, transform func([]byte) []byte) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if transform != nil {
		data = transform(data)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, fs.FileMode(0o644))
}

func rewriteGoSource(component thirdparty.Component, data []byte) []byte {
	rewritten := bytes.ReplaceAll(data, []byte(`"`+upstreamPrefix), []byte(`"`+localPrefix))
	for _, replacement := range stdlibReplacements {
		rewritten = bytes.ReplaceAll(rewritten, []byte(replacement[0]), []byte(replacement[1]))
	}
	result := append([]byte(header(component)), rewritten...)
	// Renaming imports can leave a group unsorted, so format. A copy that no longer parses is
	// still written, so the build reports the real problem.
	if formatted, err := format.Source(result); err == nil {
		return formatted
	}
	return result
}

func header(component thirdparty.Component) string {
	var out strings.Builder
	fmt.Fprintf(&out, "// Code copied from %s@%s.\n", component.Module, component.Version)
	fmt.Fprintf(&out, "// Upstream licence: %s (see %s/LICENSE and NOTICE.txt).\n", component.License, packageName)
	out.WriteString("// Modified by tools/tsgo-sync:\n")
	for _, modification := range component.Modifications {
		fmt.Fprintf(&out, "//   - %s\n", modification)
	}
	out.WriteString("\n")
	return out.String()
}

func writeReadme(component thirdparty.Component, target string) error {
	var out strings.Builder
	out.WriteString("# Vendored TypeScript parser\n\n")
	fmt.Fprintf(&out, "A copy of the parser, scanner and AST packages of %s at version `%s`.\n\n",
		component.Name, component.Version)
	out.WriteString("Do not edit these files by hand. Refresh them with:\n\n")
	fmt.Fprintf(&out, "    %s\n\n", component.SyncTool)
	out.WriteString("Upstream keeps these packages under `internal/`, which Go does not allow other\nmodules to import. Copying them is permitted by the Apache License 2.0; the\nLICENSE and NOTICE.txt files are reproduced here and every copied file carries\na modification notice.\n\n")
	out.WriteString("## Modifications\n\n")
	for _, modification := range component.Modifications {
		fmt.Fprintf(&out, "- %s\n", modification)
	}
	fmt.Fprintf(&out, "\nThe pinned version is recorded in `internal/thirdparty/components.go` and\nchecked against the Go vulnerability database by `go run ./tools/third-party-audit`.\n")
	return os.WriteFile(filepath.Join(target, "README.md"), []byte(out.String()), 0o644)
}
