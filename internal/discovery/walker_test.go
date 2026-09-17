package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/config"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
	"github.com/lumioguard/lumioguard-cc/internal/language"
)

func writeFile(t *testing.T, root, relative string) {
	t.Helper()
	absolute := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absolute, []byte("export const x = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverSelectsIncludedFilesAndPrunesExcludedDirectories(t *testing.T) {
	root := t.TempDir()
	for _, file := range []string{
		"src/b.ts",
		"src/a.tsx",
		"src/readme.md",
		"node_modules/pkg/index.js",
		"dist/bundle.js",
		"lib/app.min.js",
		".lumioguard-cc/baselines/initial.json",
		"tests/unit.test.ts",
	} {
		writeFile(t, root, file)
	}
	source := domain.SourceConfig{
		Include: []string{"**/*.ts", "**/*.tsx", "**/*.js"},
		Exclude: []string{"**/dist/**", "**/*.min.js"},
	}
	result, err := NewWalker().Discover(root, source)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(root, "src", "a.tsx"),
		filepath.Join(root, "src", "b.ts"),
		filepath.Join(root, "tests", "unit.test.ts"),
	}
	if len(result.Files) != len(want) {
		t.Fatalf("files = %v, want %v", result.Files, want)
	}
	for i := range want {
		if result.Files[i] != want[i] {
			t.Fatalf("files[%d] = %q, want %q", i, result.Files[i], want[i])
		}
	}
	// node_modules and .lumioguard-cc are hard-skipped; dist is excluded by pattern.
	if result.Exclusions.ExcludedDirectories != 3 {
		t.Fatalf("excluded directories = %d, want 3", result.Exclusions.ExcludedDirectories)
	}
	if result.Exclusions.ExcludedFiles != 1 {
		t.Fatalf("excluded files = %d, want 1", result.Exclusions.ExcludedFiles)
	}
}

// TestDiscoversEveryExtensionAnAdapterAccepts checks that the default include
// patterns cover every extension the language table declares.
func TestDiscoversEveryExtensionAnAdapterAccepts(t *testing.T) {
	root := t.TempDir()
	var want []string
	for _, item := range language.All() {
		for _, extension := range item.Extensions {
			name := "sample" + extension
			writeFile(t, root, name)
			want = append(want, name)
		}
	}
	result, err := NewWalker().Discover(root, config.Default().Source)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != len(want) {
		t.Fatalf("discovered %d files, want %d (%v)", len(result.Files), len(want), want)
	}
}
