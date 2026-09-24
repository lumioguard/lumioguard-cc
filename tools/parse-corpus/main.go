// Command parse-corpus parses a directory of real sources and reports failures and timing.
// With -python it also compares function boundaries with CPython's ast module.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/cfamily"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/golang"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/java"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/python"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/typescript"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/paths"
	"github.com/lumioguard/lumioguard-cc/internal/report"
)

type failure struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

type corpusReport struct {
	Language     string    `json:"language"`
	Directory    string    `json:"directory"`
	Files        int       `json:"files"`
	Parsed       int       `json:"parsed"`
	Failed       int       `json:"failed"`
	Functions    int       `json:"functions"`
	Bytes        int64     `json:"bytes"`
	DurationMs   int64     `json:"durationMs"`
	FilesPerSec  float64   `json:"filesPerSecond"`
	MBPerSec     float64   `json:"megabytesPerSecond"`
	Failures     []failure `json:"failures"`
	Differential *diff     `json:"differential,omitempty"`
}

type diff struct {
	FilesCompared int      `json:"filesCompared"`
	FilesEqual    int      `json:"filesEqual"`
	FilesDiffer   int      `json:"filesDiffer"`
	Examples      []string `json:"examples"`
}

type functionKey struct {
	Name    string `json:"name"`
	Line    int    `json:"line"`
	EndLine int    `json:"endLine"`
	Params  int    `json:"params"`
}

func main() {
	language := flag.String("lang", "python", "python, java, typescript, go or cfamily")
	directory := flag.String("dir", "", "directory to scan")
	limit := flag.Int("limit", 0, "maximum number of files (0 = all)")
	pythonExe := flag.String("python", "", "python executable for a differential check of function boundaries")
	maxFailures := flag.Int("failures", 15, "number of failures to list")
	flag.Parse()
	if *directory == "" {
		fmt.Fprintln(os.Stderr, "-dir is required")
		os.Exit(2)
	}
	if err := run(*language, *directory, *limit, *pythonExe, *maxFailures); err != nil {
		fmt.Fprintln(os.Stderr, "parse-corpus:", err)
		os.Exit(1)
	}
}

func adapterFor(language string) (adapter.LanguageAdapter, error) {
	switch language {
	case "python":
		return python.New(), nil
	case "java":
		return java.New(), nil
	case "typescript":
		return typescript.New(), nil
	case "go":
		return golang.New(), nil
	case "cfamily":
		return cfamily.New(), nil
	default:
		return nil, fmt.Errorf("unknown language %q", language)
	}
}

func run(language, directory string, limit int, pythonExe string, maxFailures int) error {
	languageAdapter, err := adapterFor(language)
	if err != nil {
		return err
	}
	var files []string
	err = filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == "__pycache__" || entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if languageAdapter.Supports(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	out := corpusReport{Language: language, Directory: directory, Files: len(files)}
	results := make([]*adapter.SourceFile, len(files))
	started := time.Now()
	var wait sync.WaitGroup
	semaphore := make(chan struct{}, runtime.NumCPU())
	for i, file := range files {
		wait.Add(1)
		semaphore <- struct{}{}
		go func(i int, file string) {
			defer wait.Done()
			defer func() { <-semaphore }()
			defer func() {
				if recovered := recover(); recovered != nil {
					results[i] = &adapter.SourceFile{RelativePath: file, Diagnostics: []domain.Diagnostic{
						domain.RequiredError("panic", file, fmt.Sprint(recovered)),
					}}
				}
			}()
			result, err := languageAdapter.AnalyzeFile(context.Background(), directory, file)
			if err != nil {
				results[i] = &adapter.SourceFile{RelativePath: file, Diagnostics: []domain.Diagnostic{domain.RequiredError("io", file, err.Error())}}
				return
			}
			results[i] = result
		}(i, file)
	}
	wait.Wait()
	out.DurationMs = time.Since(started).Milliseconds()

	for _, result := range results {
		out.Bytes += int64(len(result.Code))
		if result.HasRequiredDiagnostic() {
			out.Failed++
			if len(out.Failures) < maxFailures {
				out.Failures = append(out.Failures, failure{File: result.RelativePath, Message: result.Diagnostics[0].Message})
			}
			continue
		}
		out.Parsed++
		out.Functions += countFunctions(result)
	}
	seconds := float64(out.DurationMs) / 1000
	if seconds > 0 {
		out.FilesPerSec = domain.Round(float64(out.Files)/seconds, 2)
		out.MBPerSec = domain.Round(float64(out.Bytes)/1024/1024/seconds, 2)
	}
	if pythonExe != "" && language == "python" {
		differential, err := comparePython(pythonExe, directory, files, results)
		if err != nil {
			return err
		}
		out.Differential = differential
	}
	return report.WriteJSON(os.Stdout, out)
}

func countFunctions(file *adapter.SourceFile) int {
	count := 0
	for _, measurement := range file.Measurements {
		if measurement.MetricID == domain.MetricCyclomatic {
			count++
		}
	}
	return count
}

// pythonScript prints one JSON object per file: its relative path and the
// (name, lineno, end_lineno, parameter count) of every def and async def.
const pythonScript = `
import ast, json, sys
for line in sys.stdin:
    path = line.rstrip("\n")
    if not path:
        continue
    try:
        with open(path, "rb") as handle:
            source = handle.read()
        tree = ast.parse(source)
    except Exception as error:
        print(json.dumps({"file": path, "error": type(error).__name__}))
        continue
    functions = []
    for node in ast.walk(tree):
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            a = node.args
            params = len(a.posonlyargs) + len(a.args) + len(a.kwonlyargs) + (1 if a.vararg else 0) + (1 if a.kwarg else 0)
            functions.append({"name": node.name, "line": node.lineno, "endLine": node.end_lineno, "params": params})
    print(json.dumps({"file": path, "functions": functions}))
`

func comparePython(pythonExe, directory string, files []string, results []*adapter.SourceFile) (*diff, error) {
	command := exec.Command(pythonExe, "-c", pythonScript)
	var stdin bytes.Buffer
	for _, file := range files {
		stdin.WriteString(file + "\n")
	}
	command.Stdin = &stdin
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", pythonExe, err)
	}
	theirs := map[string]any{}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 64*1024*1024)
	for scanner.Scan() {
		var record struct {
			File      string        `json:"file"`
			Error     string        `json:"error"`
			Functions []functionKey `json:"functions"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, err
		}
		if record.Error != "" {
			theirs[record.File] = record.Error
		} else {
			theirs[record.File] = record.Functions
		}
	}
	if err := command.Wait(); err != nil {
		return nil, fmt.Errorf("%s: %w", pythonExe, err)
	}

	result := &diff{}
	for i, file := range files {
		expected, ok := theirs[file]
		if !ok {
			continue
		}
		mine := results[i]
		if _, failed := expected.(string); failed {
			// CPython rejects the file too; both agree it is not analyzable.
			if mine.HasRequiredDiagnostic() {
				result.FilesCompared++
				result.FilesEqual++
			}
			continue
		}
		result.FilesCompared++
		ours := ourFunctions(mine)
		theirFunctions := expected.([]functionKey)
		if sameFunctions(ours, theirFunctions) {
			result.FilesEqual++
			continue
		}
		result.FilesDiffer++
		if len(result.Examples) < 15 {
			result.Examples = append(result.Examples, fmt.Sprintf("%s: ours=%s theirs=%s", paths.Relative(directory, file), describe(ours, theirFunctions), describe(theirFunctions, ours)))
		}
	}
	return result, nil
}

// ourFunctions lists def/async def functions from the adapter's measurements.
// Lambdas are excluded because CPython gives them no name.
func ourFunctions(file *adapter.SourceFile) []functionKey {
	var out []functionKey
	params := map[string]int{}
	for _, measurement := range file.Measurements {
		if measurement.MetricID == domain.MetricParameterCount {
			params[measurement.Scope.Key] = int(*measurement.Value)
		}
	}
	for _, measurement := range file.Measurements {
		if measurement.MetricID != domain.MetricCyclomatic {
			continue
		}
		symbol := measurement.Scope.Symbol
		if strings.Contains(symbol, "<lambda>") {
			continue
		}
		name := symbol
		if index := strings.LastIndex(name, "."); index >= 0 {
			name = name[index+1:]
		}
		if index := strings.Index(name, "#"); index >= 0 {
			name = name[:index]
		}
		out = append(out, functionKey{Name: name, Line: measurement.Scope.Line, EndLine: measurement.Scope.EndLine, Params: params[measurement.Scope.Key]})
	}
	return out
}

func sameFunctions(a, b []functionKey) bool {
	if len(a) != len(b) {
		return false
	}
	counts := map[functionKey]int{}
	for _, key := range a {
		counts[key]++
	}
	for _, key := range b {
		counts[key]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

// describe lists the entries of a that are missing from b.
func describe(a, b []functionKey) string {
	counts := map[functionKey]int{}
	for _, key := range b {
		counts[key]++
	}
	var missing []string
	for _, key := range a {
		if counts[key] > 0 {
			counts[key]--
			continue
		}
		missing = append(missing, fmt.Sprintf("%s@%d-%d/%d", key.Name, key.Line, key.EndLine, key.Params))
		if len(missing) == 3 {
			break
		}
	}
	return "[" + strings.Join(missing, " ") + "]"
}
