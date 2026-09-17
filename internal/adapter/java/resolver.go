package java

import (
	"path"
	"sort"
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/paths"
)

// Resolver implements adapter.ImportResolver keyed on package declarations, so any
// layout works. Implicit type references are never unresolved: most name JDK types.
type Resolver struct{}

// NewResolver creates a Resolver.
func NewResolver() *Resolver {
	return &Resolver{}
}

// Resolve implements adapter.ImportResolver.
func (r *Resolver) Resolve(source *adapter.SourceFile, imp adapter.Import, index *adapter.ModuleIndex) adapter.Resolution {
	if imp.Kind == adapter.ImportImplicit {
		return r.resolveImplicit(source, imp.Specifier, index)
	}
	return r.resolveDeclared(source, imp.Specifier, index)
}

func (r *Resolver) resolveDeclared(source *adapter.SourceFile, specifier string, index *adapter.ModuleIndex) adapter.Resolution {
	if strings.HasSuffix(specifier, ".*") {
		return adapter.External()
	}
	segments := strings.Split(specifier, ".")
	packages := packageDirectories(index)
	if target, ok := findQualified(source, segments, packages, index); ok {
		return adapter.Internal(target)
	}
	if len(segments) > 1 {
		if _, known := packages[strings.Join(segments[:len(segments)-1], ".")]; known {
			return adapter.NotResolved()
		}
	}
	return adapter.External()
}

func (r *Resolver) resolveImplicit(source *adapter.SourceFile, name string, index *adapter.ModuleIndex) adapter.Resolution {
	segments := strings.Split(name, ".")
	packages := packageDirectories(index)
	if len(segments) > 1 {
		if target, ok := findQualified(source, segments, packages, index); ok {
			return adapter.Internal(target)
		}
	}
	// Same package: every directory holding files of the file's package, which
	// includes other source sets such as test sources.
	for _, directory := range samePackageDirectories(source, packages) {
		if target, ok := findInDirectory(source, directory, segments[0], index); ok {
			return adapter.Internal(target)
		}
	}
	for _, imp := range source.Imports {
		if !strings.HasSuffix(imp.Specifier, ".*") {
			continue
		}
		for _, directory := range packages[strings.TrimSuffix(imp.Specifier, ".*")] {
			if target, ok := findInDirectory(source, directory, segments[0], index); ok {
				return adapter.Internal(target)
			}
		}
	}
	return adapter.External()
}

// findQualified matches a qualified name from its longest package prefix down,
// so nested classes and static members resolve.
func findQualified(source *adapter.SourceFile, segments []string, packages map[string][]string, index *adapter.ModuleIndex) (string, bool) {
	for length := len(segments) - 1; length >= 0; length-- {
		for _, directory := range packages[strings.Join(segments[:length], ".")] {
			if target, ok := findInDirectory(source, directory, segments[length], index); ok {
				return target, true
			}
		}
	}
	return "", false
}

// samePackageDirectories lists the file's own directory first, then the other
// directories that declare the same package.
func samePackageDirectories(source *adapter.SourceFile, packages map[string][]string) []string {
	own := paths.Dir(source.RelativePath)
	directories := []string{own}
	for _, directory := range packages[source.Module.Package] {
		if directory != own {
			directories = append(directories, directory)
		}
	}
	return directories
}

func findInDirectory(source *adapter.SourceFile, directory, className string, index *adapter.ModuleIndex) (string, bool) {
	candidate := path.Join(directory, className+".java")
	if candidate != source.RelativePath && index.Has(candidate) {
		return candidate, true
	}
	return "", false
}

// packageDirectories maps every declared package (the default package is "")
// to the sorted directories of its analyzed files.
func packageDirectories(index *adapter.ModuleIndex) map[string][]string {
	return index.Memo("java.package-directories", func() any {
		found := map[string]map[string]bool{}
		for _, file := range index.Files() {
			if !strings.HasSuffix(file.RelativePath, ".java") {
				continue
			}
			directory := paths.Dir(file.RelativePath)
			if found[file.Module.Package] == nil {
				found[file.Module.Package] = map[string]bool{}
			}
			found[file.Module.Package][directory] = true
		}
		packages := make(map[string][]string, len(found))
		for name, directories := range found {
			for directory := range directories {
				packages[name] = append(packages[name], directory)
			}
			sort.Strings(packages[name])
		}
		return packages
	}).(map[string][]string)
}
