package adapter

import (
	"sort"
	"strings"
)

// ResolutionKind classifies the outcome of resolving one import.
type ResolutionKind int

// Resolution outcomes.
const (
	// ResolvedInternal points at another analyzed file.
	ResolvedInternal ResolutionKind = iota
	// ResolvedExternal is a package, standard library or otherwise
	// out-of-scope dependency; it is counted but never an edge.
	ResolvedExternal
	// Unresolved means the import should have matched an analyzed file but
	// did not (for example a relative path to an excluded file).
	Unresolved
)

// Resolution is the result of ImportResolver.Resolve.
type Resolution struct {
	Kind   ResolutionKind
	Target string
}

// Internal builds an internal resolution.
func Internal(target string) Resolution {
	return Resolution{Kind: ResolvedInternal, Target: target}
}

// External builds an external resolution.
func External() Resolution {
	return Resolution{Kind: ResolvedExternal}
}

// NotResolved builds an unresolved resolution.
func NotResolved() Resolution {
	return Resolution{Kind: Unresolved}
}

// ImportResolver maps an import of a source file onto an analyzed file using
// the module system rules of the adapter's language.
type ImportResolver interface {
	Resolve(source *SourceFile, imp Import, index *ModuleIndex) Resolution
}

// ModuleIndex is the set of analyzed files of one run with lookups resolvers
// need. Resolvers may memoise derived data per index.
type ModuleIndex struct {
	byPath map[string]*SourceFile
	paths  []string
	memo   map[string]any
}

// NewModuleIndex builds an index over repository-relative paths.
func NewModuleIndex(files []*SourceFile) *ModuleIndex {
	index := &ModuleIndex{byPath: make(map[string]*SourceFile, len(files)), memo: map[string]any{}}
	for _, file := range files {
		index.byPath[file.RelativePath] = file
		index.paths = append(index.paths, file.RelativePath)
	}
	sort.Strings(index.paths)
	return index
}

// Has reports whether the path is an analyzed file.
func (i *ModuleIndex) Has(path string) bool {
	_, ok := i.byPath[path]
	return ok
}

// Paths returns every analyzed path in sorted order.
func (i *ModuleIndex) Paths() []string {
	return i.paths
}

// Files returns every analyzed file in path order.
func (i *ModuleIndex) Files() []*SourceFile {
	files := make([]*SourceFile, 0, len(i.paths))
	for _, path := range i.paths {
		files = append(files, i.byPath[path])
	}
	return files
}

// HasDirectory reports whether any analyzed file lives under the directory
// (repository-relative, without trailing slash; "" is the root).
func (i *ModuleIndex) HasDirectory(directory string) bool {
	if directory == "" || directory == "." {
		return len(i.paths) > 0
	}
	prefix := directory + "/"
	position := sort.SearchStrings(i.paths, prefix)
	return position < len(i.paths) && strings.HasPrefix(i.paths[position], prefix)
}

// Memo returns the value cached under key, computing it on first use.
func (i *ModuleIndex) Memo(key string, build func() any) any {
	if value, ok := i.memo[key]; ok {
		return value
	}
	value := build()
	i.memo[key] = value
	return value
}
