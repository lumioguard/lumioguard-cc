// Command java-grammar-sync regenerates internal/adapter/java/syntax from the pinned ANTLR Java
// grammar and stamps its BSD-3-Clause notice on every file. Needs a Java runtime.
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/paths"
	"github.com/lumioguard/lumioguard-cc/internal/thirdparty"
)

const (
	grammarRepo = "antlr/grammars-v4"
	grammarPath = "java/java"
	packageName = "syntax"
)

var grammarFiles = []string{"JavaLexer.g4", "JavaParser.g4", "Go/java_parser_base.go"}

// predicateRewrites port the grammar's semantic predicates from the Java
// target form to the Go form, as the grammar's own transformGrammar.py does.
var predicateRewrites = [][2]string{
	{"this.IsNotIdentifierAssign()", "p.IsNotIdentifierAssign()"},
	{"this.DoLastRecordComponent()", "p.DoLastRecordComponent()"},
}

// licenceBlock matches the leading block comment of a grammar file, which is
// where grammars-v4 keeps the BSD licence text.
var licenceBlock = regexp.MustCompile(`(?s)^\s*/\*(.*?)\*/`)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "java-grammar-sync:", err)
		os.Exit(1)
	}
}

func run() error {
	component := thirdparty.JavaGrammar
	root, err := thirdparty.RepositoryRoot()
	if err != nil {
		return err
	}
	runtime, err := runtimeVersion(root)
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "lumioguard-cc-java-grammar-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()

	for _, name := range grammarFiles {
		url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", grammarRepo, component.Version, grammarPath, name)
		if err := download(url, filepath.Join(work, filepath.Base(name))); err != nil {
			return err
		}
	}
	licence, err := grammarLicence(filepath.Join(work, "JavaParser.g4"))
	if err != nil {
		return err
	}
	jar := filepath.Join(work, "antlr.jar")
	jarURL := fmt.Sprintf("https://repo1.maven.org/maven2/org/antlr/antlr4/%s/antlr4-%s-complete.jar",
		thirdparty.ANTLRToolVersion, thirdparty.ANTLRToolVersion)
	if err := download(jarURL, jar); err != nil {
		return err
	}

	target := paths.Resolve(root, component.Path)
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	command := exec.Command("java", "-jar", jar, "-Dlanguage=Go", "-package", packageName, "-o", target,
		"-Xexact-output-dir", "-visitor", "-listener", "JavaLexer.g4", "JavaParser.g4")
	command.Dir = work
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("antlr: %v\n%s", err, output)
	}

	if err := stampGenerated(component, target); err != nil {
		return err
	}
	if err := assertNoVersionCheckCall(target); err != nil {
		return err
	}
	if err := copyParserBase(component, filepath.Join(work, "java_parser_base.go"), target); err != nil {
		return err
	}
	for _, name := range []string{"JavaLexer.interp", "JavaLexer.tokens", "JavaParser.interp", "JavaParser.tokens"} {
		_ = os.Remove(filepath.Join(target, name)) // absent after a clean generation
	}
	if err := os.WriteFile(filepath.Join(target, "LICENSE"), []byte(licence), 0o644); err != nil {
		return err
	}
	if err := writeReadme(component, target, runtime); err != nil {
		return err
	}
	fmt.Printf("generated the Java parser from %s@%s (%s) with ANTLR %s, runtime %s, into %s\n",
		grammarRepo, component.Version, grammarPath, thirdparty.ANTLRToolVersion, runtime, component.Path)
	return nil
}

// runtimeVersion reports the ANTLR Go runtime version linked from go.mod.
func runtimeVersion(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	pattern := regexp.MustCompile(`github\.com/antlr4-go/antlr/v4 v([0-9.]+)`)
	match := pattern.FindSubmatch(data)
	if match == nil {
		return "", fmt.Errorf("could not find github.com/antlr4-go/antlr/v4 in go.mod")
	}
	return string(match[1]), nil
}

// assertNoVersionCheckCall fails if generated code calls the ANTLR runtime version check, which
// would print to stdout and corrupt JSON reports because the tool is ahead of the runtime.
func assertNoVersionCheckCall(target string) error {
	files, err := filepath.Glob(filepath.Join(target, "*.go"))
	if err != nil {
		return err
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte("heckVersion(")) {
			return fmt.Errorf("%s calls the ANTLR runtime version check, which prints to stdout on "+
				"mismatch and would corrupt JSON output; align ANTLRToolVersion with the runtime in go.mod",
				filepath.Base(path))
		}
	}
	return nil
}

func grammarLicence(grammarFile string) (string, error) {
	data, err := os.ReadFile(grammarFile)
	if err != nil {
		return "", err
	}
	match := licenceBlock.FindSubmatch(data)
	if match == nil {
		return "", fmt.Errorf("no licence block found in %s", filepath.Base(grammarFile))
	}
	body := strings.TrimSpace(string(match[1]))
	var out strings.Builder
	out.WriteString("The Java grammar this parser is generated from is distributed under the\n")
	out.WriteString("following licence. It is reproduced here because BSD-3-Clause requires\n")
	out.WriteString("source redistributions to retain the copyright notice, the list of\n")
	out.WriteString("conditions and the disclaimer.\n\n")
	out.WriteString("Source: https://github.com/" + grammarRepo + "/tree/" + thirdparty.JavaGrammar.Version + "/" + grammarPath + "\n\n")
	for _, line := range strings.Split(body, "\n") {
		out.WriteString(strings.TrimRight(line, " \t") + "\n")
	}
	return out.String(), nil
}

// stampGenerated inserts the grammar attribution after the "Code generated"
// line, which Go tooling requires to stay first.
func stampGenerated(component thirdparty.Component, target string) error {
	entries, err := filepath.Glob(filepath.Join(target, "*.go"))
	if err != nil {
		return err
	}
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, rewrite := range predicateRewrites {
			data = bytes.ReplaceAll(data, []byte(rewrite[0]), []byte(rewrite[1]))
		}
		line, rest, found := bytes.Cut(data, []byte("\n"))
		if !found || !bytes.HasPrefix(line, []byte("// Code generated")) {
			return fmt.Errorf("%s does not start with a generated-code marker", filepath.Base(path))
		}
		stamped := append(append(append([]byte{}, line...), '\n'), []byte(attribution(component))...)
		if err := os.WriteFile(path, append(stamped, rest...), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func copyParserBase(component thirdparty.Component, source, target string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	renamed := strings.Replace(string(data), "package parser", "package "+packageName, 1)
	stamped := "// Code copied from the ANTLR grammars-v4 Java grammar. DO NOT EDIT.\n" +
		attribution(component) + "\n" + renamed
	return os.WriteFile(filepath.Join(target, "java_parser_base.go"), []byte(stamped), 0o644)
}

func attribution(component thirdparty.Component) string {
	var out strings.Builder
	out.WriteString("//\n")
	fmt.Fprintf(&out, "// Generated from %s, commit %s.\n", component.Name, component.Version)
	fmt.Fprintf(&out, "// Grammar copyright (c) 2013 Terence Parr, Sam Harwell; (c) 2017 Ivan Kochurkin;\n")
	fmt.Fprintf(&out, "// (c) 2021, 2022 Michal Lorek. Licensed %s: see LICENSE in this directory for\n", component.License)
	out.WriteString("// the full notice, conditions and disclaimer.\n")
	fmt.Fprintf(&out, "// Regenerate with: %s\n", component.SyncTool)
	return out.String()
}

func download(url, target string) error {
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, response.StatusCode)
	}
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, response.Body); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func writeReadme(component thirdparty.Component, target, runtime string) error {
	var out strings.Builder
	out.WriteString("# Generated Java parser\n\n")
	out.WriteString("Generated code. Do not edit by hand; regenerate with:\n\n")
	fmt.Fprintf(&out, "    %s\n\n", component.SyncTool)
	fmt.Fprintf(&out, "Source grammar: %s, path `%s`, commit `%s`.\n", grammarRepo, grammarPath, component.Version)
	fmt.Fprintf(&out, "Licence: %s, reproduced in full in LICENSE beside this file.\n", component.License)
	fmt.Fprintf(&out, "Generator: ANTLR %s. Runtime: `github.com/antlr4-go/antlr/v4 v%s`, the latest\npublished Go runtime. The tool is deliberately ahead; the sync tool asserts that\nthe generated code never calls the runtime's version check, which is the only\nway the difference could become observable.\n\n",
		thirdparty.ANTLRToolVersion, runtime)
	out.WriteString("## Modifications\n\n")
	for _, modification := range component.Modifications {
		fmt.Fprintf(&out, "- %s\n", modification)
	}
	out.WriteString("\nThe pinned commit is recorded in `internal/thirdparty/components.go`.\n")
	return os.WriteFile(filepath.Join(target, "README.md"), []byte(out.String()), 0o644)
}
