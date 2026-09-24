package cfamily

import (
	"path"
	"sort"
	"strings"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/paths"
)

// includeResolver approximates the compiler's include search without include paths, and
// never picks between two equally good files: an ambiguous include is unresolved.
type includeResolver struct{}

// Resolve looks a quoted include up next to the file, then from the root and as the one
// path ending in the name, as an include path would; a bare <name.h> is external.
func (includeResolver) Resolve(source *adapter.SourceFile, imp adapter.Import, index *adapter.ModuleIndex) adapter.Resolution {
	spelled := imp.Specifier
	if len(spelled) < 3 {
		return adapter.External()
	}
	quoted := spelled[0] == '"'
	name := path.Clean(strings.ReplaceAll(spelled[1:len(spelled)-1], "\\", "/"))
	if quoted {
		if sibling := path.Join(paths.Dir(source.RelativePath), name); isTarget(sibling, source, index) {
			return adapter.Internal(sibling)
		}
	}
	if !quoted && !strings.Contains(name, "/") {
		return adapter.External()
	}
	if isTarget(name, source, index) {
		return adapter.Internal(name)
	}
	var candidates []string
	for _, candidate := range suffixIndex(index)[name] {
		if candidate != source.RelativePath {
			candidates = append(candidates, candidate)
		}
	}
	switch len(candidates) {
	case 0:
		return adapter.External()
	case 1:
		return adapter.Internal(candidates[0])
	default:
		return adapter.NotResolved()
	}
}

// isTarget reports an analyzed C or C++ file other than the including one.
func isTarget(candidate string, source *adapter.SourceFile, index *adapter.ModuleIndex) bool {
	return candidate != source.RelativePath && supportedExtensions[paths.Extension(candidate)] && index.Has(candidate)
}

// suffixIndex maps every trailing path of each analyzed C or C++ file, such as
// "b/c.h" and "c.h" for "a/b/c.h", to the files ending in it.
func suffixIndex(index *adapter.ModuleIndex) map[string][]string {
	return index.Memo("cfamily.suffixes", func() any {
		suffixes := map[string][]string{}
		for _, file := range index.Paths() {
			if !supportedExtensions[paths.Extension(file)] {
				continue
			}
			for rest := file; ; {
				suffixes[rest] = append(suffixes[rest], file)
				cut := strings.IndexByte(rest, '/')
				if cut < 0 {
					break
				}
				rest = rest[cut+1:]
			}
		}
		for _, files := range suffixes {
			sort.Strings(files)
		}
		return suffixes
	}).(map[string][]string)
}
