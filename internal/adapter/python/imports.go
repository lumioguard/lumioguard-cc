package python

import (
	"strings"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/python/syntax"
)

// collectImports gathers imports, from-imports and literal dynamic imports.
// Relative specifiers keep their leading dots; star imports end in ".*".
// The spans cover the import and from-import statements.
func collectImports(module *syntax.Node) ([]adapter.Import, []adapter.LineSpan) {
	imports := []adapter.Import{}
	var spans []adapter.LineSpan
	syntax.Walk(module, func(node *syntax.Node) bool {
		switch node.Kind {
		case syntax.Import:
			spans = append(spans, adapter.LineSpan{Line: node.Line, EndLine: node.EndLine})
			for _, alias := range node.Names {
				imports = append(imports, adapter.Import{Specifier: alias.Name, Line: node.Line, Kind: adapter.ImportStatic})
			}
		case syntax.ImportFrom:
			spans = append(spans, adapter.LineSpan{Line: node.Line, EndLine: node.EndLine})
			prefix := strings.Repeat(".", node.Level) + node.Module
			for _, alias := range node.Names {
				specifier := prefix
				if node.Module != "" {
					specifier += "."
				}
				specifier += alias.Name
				imports = append(imports, adapter.Import{Specifier: specifier, Line: node.Line, Kind: adapter.ImportStatic})
			}
		case syntax.Call:
			if specifier, ok := dynamicImport(node); ok {
				imports = append(imports, adapter.Import{Specifier: specifier, Line: node.Line, Kind: adapter.ImportDynamic})
			}
		}
		return true
	})
	return imports, spans
}

func dynamicImport(call *syntax.Node) (string, bool) {
	if len(call.Values) == 0 {
		return "", false
	}
	callee := call.Func
	isImporter := false
	switch {
	case callee.Kind == syntax.NameNode && callee.Name == "__import__":
		isImporter = true
	case callee.Kind == syntax.Attribute && callee.Name == "import_module" && callee.Object != nil &&
		callee.Object.Kind == syntax.NameNode && callee.Object.Name == "importlib":
		isImporter = true
	}
	if !isImporter {
		return "", false
	}
	return constantString(call.Values[0])
}

// constantString extracts the text of a plain (non-raw, non-formatted,
// non-bytes) single-token string literal.
func constantString(node *syntax.Node) (string, bool) {
	if node == nil || node.Kind != syntax.Constant {
		return "", false
	}
	text := node.Value
	if len(text) < 2 {
		return "", false
	}
	quote := text[0]
	if quote != '\'' && quote != '"' {
		return "", false
	}
	if strings.HasPrefix(text, string(quote)+string(quote)+string(quote)) {
		if len(text) < 6 {
			return "", false
		}
		return text[3 : len(text)-3], true
	}
	if text[len(text)-1] != quote {
		return "", false
	}
	return text[1 : len(text)-1], true
}
