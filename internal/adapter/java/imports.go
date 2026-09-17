package java

import (
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"

	"github.com/lumiostack/lumioguard-cc/internal/adapter"
	"github.com/lumiostack/lumioguard-cc/internal/adapter/java/syntax"
)

func packageName(p *parsedFile) string {
	unit, ok := p.tree.(*syntax.CompilationUnitContext)
	if !ok || unit.PackageDeclaration() == nil {
		return ""
	}
	return qualifiedName(unit.PackageDeclaration().QualifiedName())
}

func qualifiedName(name syntax.IQualifiedNameContext) string {
	if name == nil {
		return ""
	}
	return joinIdentifiers(name.AllIdentifier())
}

func joinIdentifiers(identifiers []syntax.IIdentifierContext) string {
	parts := make([]string, 0, len(identifiers))
	for _, identifier := range identifiers {
		parts = append(parts, identifier.GetText())
	}
	return strings.Join(parts, ".")
}

// typeReference is a type name used in the file: a simple name such as
// "Helper" or a qualified one such as "com.acme.util.Helper" or "Outer.Inner".
type typeReference struct {
	name string
	line int
}

// collectImports gathers declared imports plus implicit type references, which
// let same-package and wildcard-imported dependencies appear in the graph.
// The spans cover the import declarations only.
func collectImports(p *parsedFile) ([]adapter.Import, []adapter.LineSpan) {
	imports := []adapter.Import{}
	var spans []adapter.LineSpan
	covered := map[string]bool{}
	unit, ok := p.tree.(*syntax.CompilationUnitContext)
	if ok {
		for _, declaration := range unit.AllImportDeclaration() {
			spans = append(spans, adapter.LineSpan{Line: lineOf(declaration), EndLine: endLineOf(declaration)})
			specifier := qualifiedName(declaration.QualifiedName())
			if declaration.MUL() != nil {
				specifier += ".*"
			} else if declaration.STATIC() == nil {
				covered[lastSegment(specifier)] = true
			}
			covered[specifier] = true
			imports = append(imports, adapter.Import{Specifier: specifier, Line: lineOf(declaration), Kind: adapter.ImportStatic})
		}
	}
	declared := declaredTypeNames(p.tree)
	firstLine := map[string]int{}
	walkTree(p.tree, func(node antlr.Tree) {
		for _, reference := range typeReferences(node) {
			simple := lastSegment(reference.name)
			if !isTypeName(simple) || covered[reference.name] {
				continue
			}
			if !strings.Contains(reference.name, ".") && (declared[simple] || covered[simple]) {
				continue
			}
			if line, seen := firstLine[reference.name]; !seen || reference.line < line {
				firstLine[reference.name] = reference.line
			}
		}
	})
	names := slices.Sorted(maps.Keys(firstLine))
	for _, name := range names {
		imports = append(imports, adapter.Import{Specifier: name, Line: firstLine[name], Kind: adapter.ImportImplicit})
	}
	return imports, spans
}

// typeReferences returns the type names one node uses: class, catch and created
// types, uppercase primary identifiers and annotations.
func typeReferences(node antlr.Tree) []typeReference {
	var out []typeReference
	add := func(ctx antlr.ParserRuleContext, name string) {
		if name != "" {
			out = append(out, typeReference{name: name, line: lineOf(ctx)})
		}
	}
	switch ctx := node.(type) {
	case *syntax.ClassTypeContext:
		var parts []string
		for _, packagePart := range ctx.AllPackageName() {
			parts = append(parts, joinIdentifiers(packagePart.AllIdentifier()))
		}
		for _, identifier := range ctx.AllTypeIdentifier() {
			parts = append(parts, identifier.GetText())
		}
		add(ctx, strings.Join(parts, "."))
	case *syntax.CatchTypeContext:
		for _, name := range ctx.AllQualifiedName() {
			add(ctx, qualifiedName(name))
		}
	case *syntax.CreatedNameContext:
		add(ctx, joinIdentifiers(ctx.AllIdentifier()))
	case *syntax.PrimaryContext:
		if identifier := ctx.Identifier(); identifier != nil {
			add(ctx, identifier.GetText())
		}
	case *syntax.AnnotationContext:
		add(ctx, qualifiedName(ctx.QualifiedName()))
	}
	return out
}

// declaredTypeNames lists every type declared in the file, including nested
// and local types.
func declaredTypeNames(tree antlr.Tree) map[string]bool {
	names := map[string]bool{}
	walkTree(tree, func(node antlr.Tree) {
		switch ctx := node.(type) {
		case *syntax.ClassDeclarationContext:
			names[identifierText(ctx.Identifier())] = true
		case *syntax.InterfaceDeclarationContext:
			names[identifierText(ctx.Identifier())] = true
		case *syntax.EnumDeclarationContext:
			names[identifierText(ctx.Identifier())] = true
		case *syntax.RecordDeclarationContext:
			names[identifierText(ctx.Identifier())] = true
		case *syntax.AnnotationTypeDeclarationContext:
			names[identifierText(ctx.Identifier())] = true
		}
	})
	return names
}

func walkTree(node antlr.Tree, visit func(antlr.Tree)) {
	if node == nil {
		return
	}
	visit(node)
	for _, child := range node.GetChildren() {
		walkTree(child, visit)
	}
}

func isTypeName(name string) bool {
	first, _ := utf8.DecodeRuneInString(name)
	return first != utf8.RuneError && unicode.IsUpper(first)
}

func lastSegment(specifier string) string {
	if index := strings.LastIndex(specifier, "."); index >= 0 {
		return specifier[index+1:]
	}
	return specifier
}
