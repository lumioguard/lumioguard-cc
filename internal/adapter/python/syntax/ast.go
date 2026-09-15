package syntax

// NodeKind classifies an AST node. The set mirrors CPython's ast module where
// the adapter needs the distinction; other constructs share generic kinds.
type NodeKind int

// Node kinds.
const (
	ModuleNode NodeKind = iota
	FunctionDef
	Lambda
	ClassDef
	If
	For
	While
	Try
	ExceptHandler
	With
	Match
	MatchCase
	Return
	Raise
	Assert
	Import
	ImportFrom
	Global
	Nonlocal
	Pass
	Break
	Continue
	Delete
	ExprStmt
	Assign
	AugAssign
	AnnAssign
	TypeAlias
	BoolOp
	BinOp
	UnaryOp
	IfExp
	Compare
	Call
	Attribute
	Subscript
	Slice
	Starred
	NameNode
	Constant
	List
	Tuple
	Dict
	Set
	ListComp
	SetComp
	DictComp
	GeneratorExp
	Comprehension
	Await
	Yield
	YieldFrom
	NamedExpr
	Keyword
)

// Alias is one imported name with its optional binding.
type Alias struct {
	Name   string
	AsName string
}

// Node is a CPython-like AST node with positions. Fields are shared across kinds;
// each field's comment names the kinds that use it.
type Node struct {
	Kind    NodeKind
	Line    int
	EndLine int
	Start   int
	End     int

	Name     string  // FunctionDef, Lambda, ClassDef, ExceptHandler, Attribute
	Op       string  // BoolOp ("and", "or"), BinOp
	Value    string  // Constant: source text of the first literal token
	Module   string  // ImportFrom
	Level    int     // ImportFrom: leading dots
	Params   int     // FunctionDef, Lambda: parameter slots
	Async    bool    // FunctionDef, For, With, Comprehension
	Elif     bool    // If written as elif
	Wildcard bool    // MatchCase: bare case _ without a guard
	ElseLine int     // If, For, While, Try
	OpLines  []int   // BoolOp: operator lines
	Names    []Alias // Import, ImportFrom

	Test       *Node   // If, While, IfExp; ExceptHandler type; Match subject
	Target     *Node   // For, Comprehension
	Iter       *Node   // For, Comprehension
	Left       *Node   // Compare, BinOp
	Func       *Node   // Call
	Object     *Node   // Attribute, Subscript
	Guard      *Node   // MatchCase
	Pattern    *Node   // MatchCase
	Body       []*Node // Blocks; IfExp value; comprehension element, or key and value
	OrElse     []*Node // If, For, While, Try; IfExp alternative
	Handlers   []*Node // Try: ExceptHandler nodes
	Final      []*Node // Try: finally block
	Cases      []*Node // Match: MatchCase nodes
	Values     []*Node // Defaults and annotations, bases, with items, operands, arguments, guards
	Generators []*Node // Comprehensions
	Decorators []*Node // FunctionDef, ClassDef
}

// Children returns the child nodes in a stable, roughly source order.
func (n *Node) Children() []*Node {
	var out []*Node
	add := func(nodes ...*Node) {
		for _, node := range nodes {
			if node != nil {
				out = append(out, node)
			}
		}
	}
	add(n.Decorators...)
	add(n.Pattern, n.Guard, n.Test, n.Target, n.Iter, n.Left, n.Func, n.Object)
	add(n.Values...)
	add(n.Body...)
	add(n.Generators...)
	add(n.OrElse...)
	add(n.Handlers...)
	add(n.Final...)
	add(n.Cases...)
	return out
}

// Walk visits n and its descendants depth-first; visit returns false to skip
// a subtree.
func Walk(n *Node, visit func(node *Node) bool) {
	if n == nil || !visit(n) {
		return
	}
	for _, child := range n.Children() {
		Walk(child, visit)
	}
}

// IsFunctionLike reports whether the node is a def, async def or lambda.
func (n *Node) IsFunctionLike() bool {
	return n.Kind == FunctionDef || n.Kind == Lambda
}
