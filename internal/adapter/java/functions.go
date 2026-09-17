package java

import (
	"github.com/antlr4-go/antlr/v4"

	"github.com/lumioguard/lumioguard-cc/internal/adapter/java/syntax"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
)

// collectFunctions finds every method, constructor and lambda with a body,
// derives its declaration context and builds its control-flow tree.
func collectFunctions(p *parsedFile, relativePath string) []structure.Record {
	builder := flowBuilder{p: p}
	var functions []*structure.Function
	var ancestors []antlr.Tree
	var walk func(node antlr.Tree)
	walk = func(node antlr.Tree) {
		if ctx, ok := node.(antlr.ParserRuleContext); ok {
			if name, body, parameters, isFunction := functionParts(ctx); isFunction && body != nil {
				span := declarationSpan(ctx, ancestors)
				functions = append(functions, &structure.Function{
					Name:          name,
					ClassName:     enclosingClassName(ancestors),
					EnclosingName: enclosingFunctionName(ancestors),
					Line:          lineOf(span),
					EndLine:       endLineOf(ctx),
					Start:         p.startOffset(span),
					End:           p.endOffset(ctx),
					Parameters:    parameters,
					Body:          builder.body(body),
				})
			}
		}
		ancestors = append(ancestors, node)
		for _, child := range node.GetChildren() {
			walk(child)
		}
		ancestors = ancestors[:len(ancestors)-1]
	}
	walk(p.tree)
	return structure.AssignSymbols(relativePath, functions)
}

// functionParts extracts the name, body and parameter count of a function-like
// context. Methods without a body are not functions.
func functionParts(ctx antlr.ParserRuleContext) (name string, body antlr.Tree, parameters int, ok bool) {
	switch node := ctx.(type) {
	case *syntax.MethodDeclarationContext:
		return identifierText(node.Identifier()), blockOf(node.MethodBody()), countParameters(node.FormalParameters()), true
	case *syntax.InterfaceCommonBodyDeclarationContext:
		return identifierText(node.Identifier()), blockOf(node.MethodBody()), countParameters(node.FormalParameters()), true
	case *syntax.ConstructorDeclarationContext:
		return "<init>", nonNilTree(node.GetConstructorBody()), countParameters(node.FormalParameters()), true
	case *syntax.CompactConstructorDeclarationContext:
		return "<init>", nonNilTree(node.GetConstructorBody()), 0, true
	case *syntax.LambdaExpressionContext:
		return "<lambda>", nonNilTree(node.LambdaBody()), countLambdaParameters(node.LambdaParameters()), true
	}
	return "", nil, 0, false
}

func blockOf(body syntax.IMethodBodyContext) antlr.Tree {
	if body == nil {
		return nil
	}
	return nonNilTree(body.Block())
}

// nonNilTree converts a possibly nil interface value into a nil antlr.Tree.
func nonNilTree(tree antlr.Tree) antlr.Tree {
	if tree == nil {
		return nil
	}
	if ctx, ok := tree.(antlr.ParserRuleContext); ok && ctx == nil {
		return nil
	}
	return tree
}

func countParameters(parameters syntax.IFormalParametersContext) int {
	if parameters == nil {
		return 0
	}
	count := 0
	if parameters.FormalParameter() != nil {
		count++
	}
	for _, list := range parameters.AllFormalParameterList() {
		count += len(list.AllFormalParameter())
	}
	return count
}

func countLambdaParameters(parameters syntax.ILambdaParametersContext) int {
	if parameters == nil {
		return 0
	}
	if list := parameters.FormalParameterList(); list != nil {
		return len(list.AllFormalParameter())
	}
	if list := parameters.LambdaLVTIList(); list != nil {
		return len(list.AllLambdaLVTIParameter())
	}
	return len(parameters.AllIdentifier())
}

func identifierText(identifier syntax.IIdentifierContext) string {
	if identifier == nil {
		return "<anonymous>"
	}
	return identifier.GetText()
}

// declarationSpan returns the outermost member-declaration ancestor of a
// method or constructor so that annotations and modifiers belong to it.
func declarationSpan(ctx antlr.ParserRuleContext, ancestors []antlr.Tree) antlr.ParserRuleContext {
	span := ctx
	for index := len(ancestors) - 1; index >= 0; index-- {
		switch parent := ancestors[index].(type) {
		case *syntax.MemberDeclarationContext, *syntax.GenericMethodDeclarationContext, *syntax.GenericConstructorDeclarationContext,
			*syntax.InterfaceMemberDeclarationContext, *syntax.InterfaceMethodDeclarationContext, *syntax.GenericInterfaceMethodDeclarationContext:
			span = parent.(antlr.ParserRuleContext)
		case *syntax.ClassBodyDeclarationContext:
			return parent
		case *syntax.InterfaceBodyDeclarationContext:
			return parent
		default:
			return span
		}
	}
	return span
}

// enclosingClassName returns the nearest enclosing type name, including
// anonymous classes and enum constant bodies.
func enclosingClassName(ancestors []antlr.Tree) string {
	for index := len(ancestors) - 1; index >= 0; index-- {
		switch node := ancestors[index].(type) {
		case *syntax.ClassDeclarationContext:
			return identifierText(node.Identifier())
		case *syntax.InterfaceDeclarationContext:
			return identifierText(node.Identifier())
		case *syntax.EnumDeclarationContext:
			return identifierText(node.Identifier())
		case *syntax.RecordDeclarationContext:
			return identifierText(node.Identifier())
		case *syntax.AnnotationTypeDeclarationContext:
			return identifierText(node.Identifier())
		case *syntax.ClassCreatorRestContext:
			if node.ClassBody() != nil {
				return "<anonymous-class>"
			}
		case *syntax.EnumConstantContext:
			if node.ClassBody() != nil {
				return identifierText(node.Identifier())
			}
		}
	}
	return ""
}

func enclosingFunctionName(ancestors []antlr.Tree) string {
	for index := len(ancestors) - 1; index >= 0; index-- {
		if ctx, ok := ancestors[index].(antlr.ParserRuleContext); ok {
			if name, _, _, isFunction := functionParts(ctx); isFunction {
				return name
			}
		}
	}
	return ""
}
