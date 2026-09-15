package python

import (
	"path"
	"sort"
	"strings"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/paths"
)

// Resolver implements adapter.ImportResolver with Python package rules, src layouts
// included. Unmatched relative imports and imports into analyzed packages are unresolved.
type Resolver struct{}

// NewResolver creates a Resolver.
func NewResolver() *Resolver {
	return &Resolver{}
}

// Resolve implements adapter.ImportResolver.
func (r *Resolver) Resolve(source *adapter.SourceFile, imp adapter.Import, index *adapter.ModuleIndex) adapter.Resolution {
	specifier := strings.TrimSuffix(imp.Specifier, ".*")
	level := 0
	for level < len(specifier) && specifier[level] == '.' {
		level++
	}
	segments := splitDotted(specifier[level:])
	if level > 0 {
		base := paths.Dir(source.RelativePath)
		for step := 1; step < level; step++ {
			base = paths.Dir(base)
		}
		if target, ok := resolveModule(base, segments, index); ok {
			return adapter.Internal(target)
		}
		// "from . import name": the name may be defined in the package itself.
		if len(segments) == 1 {
			if candidate := joinPath(base, "__init__.py"); index.Has(candidate) {
				return adapter.Internal(candidate)
			}
		}
		return adapter.NotResolved()
	}
	if len(segments) == 0 {
		return adapter.External()
	}
	roots := candidateRoots(source, segments[0], index)
	for _, root := range roots {
		if target, ok := resolveModule(root, segments, index); ok && target != source.RelativePath {
			return adapter.Internal(target)
		}
	}
	for _, root := range roots {
		top := joinPath(root, segments[0])
		if index.HasDirectory(top) || index.Has(top+".py") {
			return adapter.NotResolved()
		}
	}
	return adapter.External()
}

func splitDotted(dotted string) []string {
	if dotted == "" {
		return nil
	}
	return strings.Split(dotted, ".")
}

// resolveModule tries the module and package forms of a dotted name, then the
// enclosing module when the last segment is a member rather than a module.
func resolveModule(root string, segments []string, index *adapter.ModuleIndex) (string, bool) {
	if len(segments) == 0 {
		candidate := joinPath(root, "__init__.py")
		if index.Has(candidate) {
			return candidate, true
		}
		return "", false
	}
	full := joinPath(root, segments...)
	for _, candidate := range []string{full + ".py", full + "/__init__.py"} {
		if index.Has(candidate) {
			return candidate, true
		}
	}
	if len(segments) > 1 {
		parent := joinPath(root, segments[:len(segments)-1]...)
		for _, candidate := range []string{parent + ".py", parent + "/__init__.py"} {
			if index.Has(candidate) {
				return candidate, true
			}
		}
	}
	return "", false
}

// candidateRoots lists directories an absolute import is tried against: non-package
// ancestors, nearest first, then directories holding its top-level package or module.
func candidateRoots(source *adapter.SourceFile, first string, index *adapter.ModuleIndex) []string {
	var roots []string
	seen := map[string]bool{}
	add := func(root string) {
		if !seen[root] {
			seen[root] = true
			roots = append(roots, root)
		}
	}
	directory := paths.Dir(source.RelativePath)
	for {
		if !index.Has(joinPath(directory, "__init__.py")) {
			add(directory)
		}
		if directory == "" {
			break
		}
		directory = paths.Dir(directory)
	}
	packageRoots := index.Memo("python.package-roots", func() any { return buildPackageRoots(index) }).(map[string][]string)
	for _, root := range packageRoots[first] {
		add(root)
	}
	return roots
}

// buildPackageRoots maps a top-level name to the sorted directories that hold it.
// A package directory is never a root: Python 3 has no implicit relative imports.
func buildPackageRoots(index *adapter.ModuleIndex) map[string][]string {
	found := map[string]map[string]bool{}
	register := func(name, root string) {
		if index.Has(joinPath(root, "__init__.py")) {
			return
		}
		if found[name] == nil {
			found[name] = map[string]bool{}
		}
		found[name][root] = true
	}
	for _, file := range index.Paths() {
		if !strings.HasSuffix(file, ".py") {
			continue
		}
		parts := strings.Split(file, "/")
		for depth := 0; depth < len(parts)-1; depth++ {
			register(parts[depth], strings.Join(parts[:depth], "/"))
		}
		register(strings.TrimSuffix(parts[len(parts)-1], ".py"), strings.Join(parts[:len(parts)-1], "/"))
	}
	result := make(map[string][]string, len(found))
	for name, roots := range found {
		for root := range roots {
			result[name] = append(result[name], root)
		}
		sort.Slice(result[name], func(i, j int) bool {
			a, b := result[name][i], result[name][j]
			if depthOf(a) != depthOf(b) {
				return depthOf(a) < depthOf(b)
			}
			return a < b
		})
	}
	return result
}

func depthOf(directory string) int {
	if directory == "" {
		return 0
	}
	return strings.Count(directory, "/") + 1
}

// joinPath joins a root and path segments. path.Join drops empty elements, so
// the repository root needs no special case.
func joinPath(root string, segments ...string) string {
	return path.Join(append([]string{root}, segments...)...)
}
