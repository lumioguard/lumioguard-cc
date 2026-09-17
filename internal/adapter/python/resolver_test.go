package python

import (
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
)

func TestResolvePythonImports(t *testing.T) {
	index := adapter.NewModuleIndex([]*adapter.SourceFile{
		{RelativePath: "src/app/__init__.py"},
		{RelativePath: "src/app/models.py"},
		{RelativePath: "src/app/services/__init__.py"},
		{RelativePath: "src/app/services/orders.py"},
		{RelativePath: "src/app/util.py"},
		{RelativePath: "tests/test_orders.py"},
		{RelativePath: "scripts/tool.py"},
		{RelativePath: "scripts/helper.py"},
	})
	resolver := NewResolver()
	cases := []struct {
		source string
		spec   string
		want   adapter.Resolution
	}{
		{"src/app/services/orders.py", ".", adapter.Internal("src/app/services/__init__.py")},
		{"src/app/services/orders.py", "..models", adapter.Internal("src/app/models.py")},
		{"src/app/services/orders.py", "..models.User", adapter.Internal("src/app/models.py")},
		{"src/app/services/orders.py", "..util.helper", adapter.Internal("src/app/util.py")},
		{"src/app/services/orders.py", ".missing", adapter.Internal("src/app/services/__init__.py")},
		{"src/app/models.py", ".util", adapter.Internal("src/app/util.py")},
		{"src/app/models.py", ".helper_function", adapter.Internal("src/app/__init__.py")},
		{"scripts/tool.py", ".nothing", adapter.NotResolved()},
		{"src/app/models.py", "util", adapter.External()},
		{"src/app/services/orders.py", "orders", adapter.External()},
		{"src/app/models.py", "app.util", adapter.Internal("src/app/util.py")},
		{"src/app/models.py", "app.services.orders", adapter.Internal("src/app/services/orders.py")},
		{"src/app/models.py", "app.services", adapter.Internal("src/app/services/__init__.py")},
		{"tests/test_orders.py", "app.services.orders.Order", adapter.Internal("src/app/services/orders.py")},
		{"tests/test_orders.py", "app.*", adapter.Internal("src/app/__init__.py")},
		{"tests/test_orders.py", "app.nonexistent", adapter.Internal("src/app/__init__.py")},
		{"tests/test_orders.py", "app.missingpkg.mod", adapter.NotResolved()},
		{"tests/test_orders.py", "os.path", adapter.External()},
		{"tests/test_orders.py", "numpy", adapter.External()},
		{"scripts/tool.py", "helper", adapter.Internal("scripts/helper.py")},
	}
	for _, tc := range cases {
		source := &adapter.SourceFile{RelativePath: tc.source}
		got := resolver.Resolve(source, adapter.Import{Specifier: tc.spec, Kind: adapter.ImportStatic}, index)
		if got != tc.want {
			t.Errorf("%s: Resolve(%q) = %+v, want %+v", tc.source, tc.spec, got, tc.want)
		}
	}
}
