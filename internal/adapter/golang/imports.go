package golang

import (
	"strconv"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
)

// collectImports gathers every import path. Go has no dynamic imports.
func collectImports(p *parsedFile) []adapter.Import {
	imports := []adapter.Import{}
	for _, spec := range p.file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		imports = append(imports, adapter.Import{Specifier: path, Line: p.line(spec.Pos()), Kind: adapter.ImportStatic})
	}
	return imports
}
