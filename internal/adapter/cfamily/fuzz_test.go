package cfamily

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzAnalyze checks that no input makes the parser panic: a panic would abort the whole check.
func FuzzAnalyze(f *testing.F) {
	seeds, _ := filepath.Glob("../../../.examples/c*/*/src/*/*.*")
	for _, seed := range seeds {
		code, _ := os.ReadFile(seed)
		f.Add(string(code), seed)
	}
	f.Add("void f() { if (a) [&]{ return x ? y : z; }(); }", "a.cpp")
	f.Fuzz(func(_ *testing.T, code, name string) {
		Analyze(name, name, code)
	})
}
