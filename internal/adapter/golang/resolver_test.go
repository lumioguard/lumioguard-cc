package golang

import (
	"testing"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
)

func TestResolveGoImports(t *testing.T) {
	app := "example.com/app"
	file := func(path, importPath, root string) *adapter.SourceFile {
		return &adapter.SourceFile{RelativePath: path, Module: adapter.Module{Package: importPath, Root: root}}
	}
	index := adapter.NewModuleIndex([]*adapter.SourceFile{
		file("main.go", app, app),
		file("internal/app/service.go", app+"/internal/app", app),
		file("internal/app/service_test.go", app+"/internal/app", app),
		file("internal/app/aaa_test.go", app+"/internal/app", app),
		file("internal/domain/model.go", app+"/internal/domain", app),
		file("internal/domain/doc.go", app+"/internal/domain", app),
		file("internal/onlytests/x_test.go", app+"/internal/onlytests", app),
		file("tools/gen/main.go", app+"/tools/gen", app+"/tools/gen"),
		{RelativePath: "scripts/loose.go"},
	})
	resolver := NewResolver()
	cases := []struct {
		source string
		spec   string
		want   adapter.Resolution
	}{
		{"main.go", app + "/internal/app", adapter.Internal("internal/app/service.go")},
		{"main.go", app + "/internal/domain", adapter.Internal("internal/domain/doc.go")},
		{"internal/app/service_test.go", app + "/internal/app", adapter.Internal("internal/app/service.go")},
		{"main.go", app + "/internal/missing", adapter.NotResolved()},
		{"main.go", app + "/internal/onlytests", adapter.NotResolved()},
		{"main.go", app + "/tools/gen", adapter.Internal("tools/gen/main.go")},
		{"main.go", "example.com/apple/x", adapter.External()},
		{"main.go", "fmt", adapter.External()},
		{"main.go", "github.com/spf13/cobra", adapter.External()},
		{"scripts/loose.go", app + "/internal/app", adapter.Internal("internal/app/service.go")},
		{"scripts/loose.go", "os", adapter.External()},
	}
	for _, tc := range cases {
		source := &adapter.SourceFile{RelativePath: tc.source}
		got := resolver.Resolve(source, adapter.Import{Specifier: tc.spec, Kind: adapter.ImportStatic}, index)
		if got != tc.want {
			t.Errorf("%s: Resolve(%q) = %+v, want %+v", tc.source, tc.spec, got, tc.want)
		}
	}
}
