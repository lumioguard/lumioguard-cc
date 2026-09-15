package glob

import "testing"

func TestMatchesAnyUsesDoublestarSemantics(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		patterns []string
		want     bool
	}{
		{"top-level file", "a.ts", []string{"**/*.ts"}, true},
		{"nested file", "src/lib/a.ts", []string{"**/*.ts"}, true},
		{"other extension", "src/a.py", []string{"**/*.ts"}, false},
		{"minified", "dist/app.min.js", []string{"**/*.min.js"}, true},
		{"generated", "src/api.generated.ts", []string{"**/*.generated.*"}, true},
		{"file inside excluded dir", "node_modules/x/index.js", []string{"**/node_modules/**"}, true},
		{"boundary prefix", "src/domain/order.ts", []string{"src/domain/**"}, true},
		{"boundary other layer", "src/infra/db.ts", []string{"src/domain/**"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MatchesAny(tc.path, tc.patterns); got != tc.want {
				t.Fatalf("MatchesAny(%q, %v) = %v, want %v", tc.path, tc.patterns, got, tc.want)
			}
		})
	}
}

func TestMatchesDirectoryPrunesExcludedDirectories(t *testing.T) {
	patterns := []string{"**/node_modules/**", "**/dist/**", "build/**"}
	for _, dir := range []string{"node_modules", "packages/app/node_modules", "dist", "src/dist", "build"} {
		if !MatchesDirectory(dir, patterns) {
			t.Fatalf("directory %q should be excluded by %v", dir, patterns)
		}
	}
	for _, dir := range []string{"src", "distribution", "builder"} {
		if MatchesDirectory(dir, patterns) {
			t.Fatalf("directory %q should not be excluded by %v", dir, patterns)
		}
	}
}

func TestValid(t *testing.T) {
	if !Valid("src/**/*.ts") {
		t.Fatal("expected a valid pattern")
	}
	if Valid("src/[") {
		t.Fatal("expected an invalid pattern")
	}
}
