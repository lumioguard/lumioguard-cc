package java

import (
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
)

func javaFile(path, pkg string, imports ...string) *adapter.SourceFile {
	file := &adapter.SourceFile{RelativePath: path, Module: adapter.Module{Package: pkg}}
	for _, specifier := range imports {
		file.Imports = append(file.Imports, adapter.Import{Specifier: specifier, Kind: adapter.ImportStatic})
	}
	return file
}

func TestResolveJavaImports(t *testing.T) {
	service := javaFile("src/main/java/com/acme/Service.java", "com.acme", "com.acme.util.*")
	files := []*adapter.SourceFile{
		service,
		javaFile("src/main/java/com/acme/Other.java", "com.acme"),
		javaFile("src/main/java/com/acme/util/Helper.java", "com.acme.util"),
		javaFile("src/main/java/com/acme/util/Strings.java", "com.acme.util"),
		javaFile("src/test/java/com/acme/ServiceTest.java", "com.acme"),
		javaFile("Loose.java", ""),
	}
	index := adapter.NewModuleIndex(files)
	resolver := NewResolver()
	cases := []struct {
		source *adapter.SourceFile
		imp    adapter.Import
		want   adapter.Resolution
	}{
		{service, adapter.Import{Specifier: "com.acme.util.Helper", Kind: adapter.ImportStatic}, adapter.Internal("src/main/java/com/acme/util/Helper.java")},
		{service, adapter.Import{Specifier: "com.acme.util.Strings.trim", Kind: adapter.ImportStatic}, adapter.Internal("src/main/java/com/acme/util/Strings.java")},
		{service, adapter.Import{Specifier: "com.acme.util.Helper.Nested", Kind: adapter.ImportStatic}, adapter.Internal("src/main/java/com/acme/util/Helper.java")},
		{service, adapter.Import{Specifier: "com.acme.util.Missing", Kind: adapter.ImportStatic}, adapter.NotResolved()},
		{service, adapter.Import{Specifier: "java.util.List", Kind: adapter.ImportStatic}, adapter.External()},
		{service, adapter.Import{Specifier: "com.acme.util.*", Kind: adapter.ImportStatic}, adapter.External()},
		{service, adapter.Import{Specifier: "Other", Kind: adapter.ImportImplicit}, adapter.Internal("src/main/java/com/acme/Other.java")},
		{service, adapter.Import{Specifier: "Strings", Kind: adapter.ImportImplicit}, adapter.Internal("src/main/java/com/acme/util/Strings.java")},
		{service, adapter.Import{Specifier: "String", Kind: adapter.ImportImplicit}, adapter.External()},
		{service, adapter.Import{Specifier: "Service", Kind: adapter.ImportImplicit}, adapter.External()},
		{files[4], adapter.Import{Specifier: "com.acme.Service", Kind: adapter.ImportStatic}, adapter.Internal("src/main/java/com/acme/Service.java")},
		{files[4], adapter.Import{Specifier: "Other", Kind: adapter.ImportImplicit}, adapter.Internal("src/main/java/com/acme/Other.java")},
		{files[4], adapter.Import{Specifier: "Other.Nested", Kind: adapter.ImportImplicit}, adapter.Internal("src/main/java/com/acme/Other.java")},
		{files[5], adapter.Import{Specifier: "Other", Kind: adapter.ImportImplicit}, adapter.External()},
		{files[5], adapter.Import{Specifier: "com.acme.Other", Kind: adapter.ImportImplicit}, adapter.Internal("src/main/java/com/acme/Other.java")},
	}
	for _, tc := range cases {
		got := resolver.Resolve(tc.source, tc.imp, index)
		if got != tc.want {
			t.Errorf("%s: Resolve(%+v) = %+v, want %+v", tc.source.RelativePath, tc.imp, got, tc.want)
		}
	}
}
