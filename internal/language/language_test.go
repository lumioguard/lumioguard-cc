package language

import (
	"slices"
	"strings"
	"testing"
)

// TestEveryExtensionIsDiscoverable fails if an accepted extension has no
// default include pattern, which would leave its files silently unanalyzed.
func TestEveryExtensionIsDiscoverable(t *testing.T) {
	patterns := IncludePatterns()
	for _, item := range All() {
		for _, extension := range item.Extensions {
			want := "**/*" + extension
			if !slices.Contains(patterns, want) {
				t.Errorf("%s extension %q has no include pattern", item.Name, extension)
			}
			if got := NameFor("sample" + extension); got != item.Name {
				t.Errorf("NameFor(%q) = %q, want %q", extension, got, item.Name)
			}
		}
	}
}

func TestNameForIsCaseInsensitiveAndFailsClosed(t *testing.T) {
	cases := map[string]string{
		"a.ts": "TypeScript", "a.MTS": "TypeScript", "a.cts": "TypeScript",
		"a.js": "JavaScript", "a.cjs": "JavaScript",
		"a.py": "Python", "a.PYW": "Python",
		"A.java": "Java",
		"a.rs":   Unknown, "a": Unknown,
	}
	for filename, want := range cases {
		if got := NameFor(filename); got != want {
			t.Errorf("NameFor(%q) = %q, want %q", filename, got, want)
		}
	}
}

func TestExtensionSetAndNames(t *testing.T) {
	set := ExtensionSet(JavaScript, TypeScript)
	for _, extension := range []string{".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts"} {
		if !set[extension] {
			t.Errorf("extension set lacks %q", extension)
		}
	}
	if set[".py"] {
		t.Error("extension set must not leak across languages")
	}
	if got := strings.Join(Names(JavaScript, TypeScript), ","); got != "JavaScript,TypeScript" {
		t.Errorf("Names = %q", got)
	}
}
