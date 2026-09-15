package typescript

import (
	"path"
	"strings"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
)

var (
	candidateExtensions = []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}
	javaScriptExtension = map[string]bool{".js": true, ".jsx": true, ".mjs": true, ".cjs": true}
)

// Resolver implements adapter.ImportResolver for relative specifiers. Package,
// workspace and tsconfig alias specifiers are external.
type Resolver struct{}

// NewResolver creates a Resolver.
func NewResolver() *Resolver {
	return &Resolver{}
}

// Resolve tries the exact path, appended extensions, index files, then TypeScript
// sources named by their compiled JavaScript path.
func (r *Resolver) Resolve(source *adapter.SourceFile, imp adapter.Import, index *adapter.ModuleIndex) adapter.Resolution {
	if !strings.HasPrefix(imp.Specifier, ".") {
		return adapter.External()
	}
	if target, ok := resolveRelative(source.RelativePath, imp.Specifier, index); ok {
		return adapter.Internal(target)
	}
	return adapter.NotResolved()
}

func resolveRelative(sourceFile, specifier string, index *adapter.ModuleIndex) (string, bool) {
	clean := specifier
	if cut := strings.IndexAny(clean, "?#"); cut >= 0 {
		clean = clean[:cut]
	}
	base := strings.TrimPrefix(path.Clean(path.Join(path.Dir(sourceFile), clean)), "./")
	candidates := []string{base}
	extension := path.Ext(base)
	switch {
	case extension == "":
		for _, suffix := range candidateExtensions {
			candidates = append(candidates, base+suffix)
		}
		for _, suffix := range candidateExtensions {
			candidates = append(candidates, base+"/index"+suffix)
		}
	case javaScriptExtension[extension]:
		stem := strings.TrimSuffix(base, extension)
		candidates = append(candidates, stem+".ts", stem+".tsx")
	}
	for _, candidate := range candidates {
		if index.Has(candidate) {
			return candidate, true
		}
	}
	return "", false
}
