package typescript

import (
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
)

func TestResolveRelativeCandidates(t *testing.T) {
	index := adapter.NewModuleIndex([]*adapter.SourceFile{
		{RelativePath: "src/a.ts"},
		{RelativePath: "src/lib/index.tsx"},
		{RelativePath: "src/util.ts"},
		{RelativePath: "src/c.js"},
	})
	source := &adapter.SourceFile{RelativePath: "src/entry.ts"}
	cases := map[string]adapter.Resolution{
		"./a":        adapter.Internal("src/a.ts"),
		"./lib":      adapter.Internal("src/lib/index.tsx"),
		"./util.js":  adapter.Internal("src/util.ts"),
		"./c.js":     adapter.Internal("src/c.js"),
		"./a?raw":    adapter.Internal("src/a.ts"),
		"../src/a":   adapter.Internal("src/a.ts"),
		"./missing":  adapter.NotResolved(),
		"./a.ts.bak": adapter.NotResolved(),
		"lodash":     adapter.External(),
		"@scope/pkg": adapter.External(),
	}
	resolver := NewResolver()
	for specifier, want := range cases {
		got := resolver.Resolve(source, adapter.Import{Specifier: specifier, Kind: adapter.ImportStatic}, index)
		if got != want {
			t.Errorf("Resolve(%q) = %+v, want %+v", specifier, got, want)
		}
	}
}
