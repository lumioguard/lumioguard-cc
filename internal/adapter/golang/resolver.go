package golang

import (
	"strings"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
)

// Resolver implements adapter.ImportResolver with Go module rules. An import
// of an analyzed package resolves to that package's representative file; an
// import inside an analyzed module that matches no package is unresolved;
// everything else, including the standard library, is external.
type Resolver struct{}

// NewResolver creates a Resolver.
func NewResolver() *Resolver {
	return &Resolver{}
}

// layout is what the resolver knows about one analysis: the representative
// file of every analyzed package and every module path seen.
type layout struct {
	packages map[string]string
	modules  []string
}

// Resolve implements adapter.ImportResolver.
func (r *Resolver) Resolve(_ *adapter.SourceFile, imp adapter.Import, index *adapter.ModuleIndex) adapter.Resolution {
	known := index.Memo("go.layout", func() any { return buildLayout(index) }).(*layout)
	if target, ok := known.packages[imp.Specifier]; ok {
		return adapter.Internal(target)
	}
	for _, module := range known.modules {
		if imp.Specifier == module || strings.HasPrefix(imp.Specifier, module+"/") {
			return adapter.NotResolved()
		}
	}
	return adapter.External()
}

// buildLayout picks, for each import path, the first file in name order that
// is not a test file. Test files never form an importable package, and one
// file per package keeps fan-out equal to the number of packages imported.
func buildLayout(index *adapter.ModuleIndex) *layout {
	known := &layout{packages: map[string]string{}}
	seen := map[string]bool{}
	for _, file := range index.Files() {
		if file.Module.Root == "" {
			continue
		}
		if !seen[file.Module.Root] {
			seen[file.Module.Root] = true
			known.modules = append(known.modules, file.Module.Root)
		}
		if strings.HasSuffix(file.RelativePath, "_test.go") {
			continue
		}
		if _, taken := known.packages[file.Module.Package]; !taken {
			known.packages[file.Module.Package] = file.RelativePath
		}
	}
	return known
}
