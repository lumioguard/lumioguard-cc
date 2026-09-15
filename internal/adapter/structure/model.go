// Package structure is the language-neutral model of functions and control flow.
// Complexity metrics are computed only here, so every language shares one definition.
package structure

// Kind classifies a node of the control-flow tree.
type Kind int

// Node kinds. Kinds that are not listed for a language simply never occur.
const (
	// KindBlock is a transparent container; it carries no complexity.
	KindBlock Kind = iota
	// KindFunction is a nested function-like construct (function, arrow, lambda, method).
	KindFunction
	// KindIf is a conditional statement; ElseIf marks a chained branch.
	KindIf
	// KindTernary is a conditional expression.
	KindTernary
	// KindSwitch is a switch or match statement; Handlers hold KindCase nodes.
	KindSwitch
	// KindCase is one case clause; Default marks the default or wildcard clause.
	KindCase
	// KindLoop is any loop; Comprehension marks a comprehension clause.
	KindLoop
	// KindTry is a try statement; Handlers hold KindCatch nodes.
	KindTry
	// KindCatch is one exception handler.
	KindCatch
	// KindJump is break or continue; Labeled marks a labelled jump.
	KindJump
	// KindLogical is a maximal sequence of binary logical operators.
	KindLogical
)

// Operator is one binary logical operator inside a KindLogical node.
type Operator struct {
	Text string
	Line int
	// Sequence reports whether the operator counts in cognitive complexity
	// sequences (&&, ||, and, or); ?? counts for cyclomatic complexity only.
	Sequence bool
}

// Node is one construct of the control-flow tree. The slots carry children
// with a semantic role; Children carries everything else in source order.
type Node struct {
	Kind  Kind
	Label string
	Line  int

	ElseIf   bool
	ElseLine int
	// HasElse records an else branch even when its body produced no nodes.
	HasElse       bool
	Default       bool
	Labeled       bool
	Comprehension bool
	Operators     []Operator

	// Condition holds tests and headers evaluated at the construct's own nesting
	// level: conditions, loop headers, catch parameters, defaults and case labels.
	Condition []*Node
	// Body holds the construct's primary body, nested one level deeper.
	Body []*Node
	// Else holds the else branch of if, ternary or loop, or a try's else block.
	Else []*Node
	// Handlers holds KindCatch nodes of a try or KindCase nodes of a switch.
	Handlers []*Node
	// Finally holds a try's finally block.
	Finally []*Node
	// Children holds the remaining children of transparent nodes.
	Children []*Node
}

// Function is a measurable function-like construct of a file.
type Function struct {
	// Name is the raw name: identifier, "<anonymous>", "<lambda>", "constructor".
	Name string
	// ClassName is the nearest enclosing class, or "".
	ClassName string
	// EnclosingName is the raw name of the nearest enclosing function, or "".
	EnclosingName string
	// Line and EndLine are 1-based source lines of the whole construct.
	Line    int
	EndLine int
	// Start and End are byte offsets from the first token to the end.
	Start int
	End   int
	// Parameters counts declared formal parameter slots.
	Parameters int
	// Body is the control-flow tree of the function body.
	Body *Node
}

// Prefix returns the symbol prefix derived from the declaration context.
func (f *Function) Prefix() string {
	prefix := ""
	if f.ClassName != "" {
		prefix = f.ClassName + "."
	}
	if f.EnclosingName != "" {
		prefix += f.EnclosingName + "."
	}
	return prefix
}

// each visits the semantic children of a node in source order.
func (n *Node) each(visit func(child *Node)) {
	if n.Kind == KindTry {
		for _, group := range [][]*Node{n.Body, n.Handlers, n.Else, n.Finally, n.Children} {
			for _, child := range group {
				visit(child)
			}
		}
		return
	}
	for _, group := range [][]*Node{n.Condition, n.Body, n.Else, n.Handlers, n.Finally, n.Children} {
		for _, child := range group {
			visit(child)
		}
	}
}

// nests reports whether the construct increases nesting depth.
func (n *Node) nests() bool {
	switch n.Kind {
	case KindIf, KindLoop:
		return !n.Comprehension
	case KindSwitch, KindCatch, KindTernary:
		return true
	default:
		return false
	}
}

// One wraps a single node, for the adapter translators that return a
// one-element list from most branches.
func One(node *Node) []*Node {
	return []*Node{node}
}
