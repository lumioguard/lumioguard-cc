package python

import (
	"slices"

	"github.com/lumioguard/lumioguard-cc/internal/adapter/python/syntax"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
)

// flowBuilder translates Python AST subtrees into the language-neutral
// control-flow model. Constructs without complexity semantics are flattened.
type flowBuilder struct{}

// body builds the root tree of a measured function: the statements of a def
// or the expression of a lambda.
func (b flowBuilder) body(function *syntax.Node) *structure.Node {
	return &structure.Node{Kind: structure.KindBlock, Children: b.convertAll(function.Body)}
}

func (b flowBuilder) convertAll(nodes []*syntax.Node) []*structure.Node {
	var out []*structure.Node
	for _, node := range nodes {
		out = append(out, b.convert(node)...)
	}
	return out
}

func (b flowBuilder) convert(node *syntax.Node) []*structure.Node {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case syntax.FunctionDef, syntax.Lambda:
		return structure.One(&structure.Node{
			Kind:      structure.KindFunction,
			Label:     "FunctionDef",
			Line:      node.Line,
			Condition: slices.Concat(b.convertAll(node.Decorators), b.convertAll(node.Values)),
			Body:      b.convertAll(node.Body),
		})
	case syntax.If:
		out := &structure.Node{
			Kind:      structure.KindIf,
			Label:     "If",
			Line:      node.Line,
			ElseIf:    node.Elif,
			ElseLine:  node.ElseLine,
			HasElse:   len(node.OrElse) > 0,
			Condition: b.convert(node.Test),
			Body:      b.convertAll(node.Body),
			Else:      b.convertAll(node.OrElse),
		}
		return structure.One(out)
	case syntax.IfExp:
		return structure.One(&structure.Node{
			Kind:      structure.KindTernary,
			Label:     "IfExp",
			Line:      node.Line,
			Condition: b.convert(node.Test),
			Body:      b.convertAll(node.Body),
			Else:      b.convertAll(node.OrElse),
		})
	case syntax.For:
		label := "For"
		if node.Async {
			label = "AsyncFor"
		}
		return structure.One(&structure.Node{
			Kind:      structure.KindLoop,
			Label:     label,
			Line:      node.Line,
			ElseLine:  node.ElseLine,
			HasElse:   len(node.OrElse) > 0,
			Condition: slices.Concat(b.convert(node.Target), b.convert(node.Iter)),
			Body:      b.convertAll(node.Body),
			Else:      b.convertAll(node.OrElse),
		})
	case syntax.While:
		return structure.One(&structure.Node{
			Kind:      structure.KindLoop,
			Label:     "While",
			Line:      node.Line,
			ElseLine:  node.ElseLine,
			HasElse:   len(node.OrElse) > 0,
			Condition: b.convert(node.Test),
			Body:      b.convertAll(node.Body),
			Else:      b.convertAll(node.OrElse),
		})
	case syntax.Try:
		out := &structure.Node{
			Kind:    structure.KindTry,
			Label:   "Try",
			Line:    node.Line,
			Body:    b.convertAll(node.Body),
			Else:    b.convertAll(node.OrElse),
			Finally: b.convertAll(node.Final),
		}
		for _, handler := range node.Handlers {
			out.Handlers = append(out.Handlers, &structure.Node{
				Kind:      structure.KindCatch,
				Label:     "ExceptHandler",
				Line:      handler.Line,
				Condition: b.convert(handler.Test),
				Body:      b.convertAll(handler.Body),
			})
		}
		return structure.One(out)
	case syntax.Match:
		out := &structure.Node{Kind: structure.KindSwitch, Label: "Match", Line: node.Line, Condition: b.convert(node.Test)}
		for _, matchCase := range node.Cases {
			out.Handlers = append(out.Handlers, &structure.Node{
				Kind:      structure.KindCase,
				Label:     "MatchCase",
				Line:      matchCase.Line,
				Default:   matchCase.Wildcard,
				Condition: slices.Concat(b.convert(matchCase.Pattern), b.convert(matchCase.Guard)),
				Children:  b.convertAll(matchCase.Body),
			})
		}
		return structure.One(out)
	case syntax.Break:
		return structure.One(&structure.Node{Kind: structure.KindJump, Label: "Break", Line: node.Line})
	case syntax.Continue:
		return structure.One(&structure.Node{Kind: structure.KindJump, Label: "Continue", Line: node.Line})
	case syntax.BoolOp:
		return structure.One(b.logical(node))
	case syntax.ListComp, syntax.SetComp, syntax.DictComp, syntax.GeneratorExp:
		return b.comprehension(node)
	default:
		return b.convertAll(node.Children())
	}
}

// logical flattens nested boolean operations into one sequence with the
// operators in source order, as the specification counts them.
func (b flowBuilder) logical(node *syntax.Node) *structure.Node {
	out := &structure.Node{Kind: structure.KindLogical, Label: "BoolOp", Line: node.Line}
	var flatten func(current *syntax.Node)
	flatten = func(current *syntax.Node) {
		if current.Kind != syntax.BoolOp {
			out.Children = append(out.Children, b.convert(current)...)
			return
		}
		for index, operand := range current.Values {
			if index > 0 {
				line := current.Line
				if index-1 < len(current.OpLines) {
					line = current.OpLines[index-1]
				}
				out.Operators = append(out.Operators, structure.Operator{Text: current.Op, Line: line, Sequence: true})
			}
			flatten(operand)
		}
	}
	flatten(node)
	return out
}

// comprehension emits one flagged loop per for clause and one flagged if per
// guard; both count for cyclomatic complexity only.
func (b flowBuilder) comprehension(node *syntax.Node) []*structure.Node {
	var out []*structure.Node
	for _, clause := range node.Generators {
		out = append(out, &structure.Node{
			Kind:          structure.KindLoop,
			Label:         "ComprehensionFor",
			Line:          clause.Line,
			Comprehension: true,
			Condition:     slices.Concat(b.convert(clause.Target), b.convert(clause.Iter)),
		})
		for index, guard := range clause.Values {
			line := clause.Line
			if index < len(clause.OpLines) {
				line = clause.OpLines[index]
			}
			out = append(out, &structure.Node{
				Kind:          structure.KindIf,
				Label:         "ComprehensionIf",
				Line:          line,
				Comprehension: true,
				Condition:     b.convert(guard),
			})
		}
	}
	return append(out, b.convertAll(node.Body)...)
}
