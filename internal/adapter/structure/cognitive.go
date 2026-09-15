package structure

// Increment is one recorded cognitive-complexity increment.
type Increment struct {
	Type    string `json:"type"`
	Line    int    `json:"line"`
	Nesting int    `json:"nesting"`
}

// CognitiveResult is the specification value with every increment.
type CognitiveResult struct {
	Value      int
	Increments []Increment
}

// Cognitive implements the Cognitive Complexity specification (SonarSource v1.5).
// The rules and deviations are in .documentations/rules/complexity.md#cognitive-complexity.
func Cognitive(function *Function) CognitiveResult {
	walker := &cognitiveWalker{result: CognitiveResult{Increments: []Increment{}}}
	walker.visitAll(function.Body.Children, 0)
	walker.visitAll(function.Body.Condition, 0)
	return walker.result
}

type cognitiveWalker struct {
	result CognitiveResult
}

func (w *cognitiveWalker) add(label string, line, nesting int) {
	w.result.Value += 1 + nesting
	w.result.Increments = append(w.result.Increments, Increment{Type: label, Line: line, Nesting: nesting})
}

func (w *cognitiveWalker) visitAll(nodes []*Node, nesting int) {
	for _, node := range nodes {
		w.visit(node, nesting)
	}
}

func (w *cognitiveWalker) visit(node *Node, nesting int) {
	switch node.Kind {
	case KindFunction:
		w.visitAll(node.Condition, nesting)
		w.visitAll(node.Body, nesting+1)
		w.visitAll(node.Children, nesting+1)
	case KindIf:
		if node.Comprehension {
			w.transparent(node, nesting)
			return
		}
		w.visitIf(node, nesting, false)
	case KindTernary:
		w.add(node.Label, node.Line, nesting)
		w.visitAll(node.Condition, nesting)
		w.visitAll(node.Body, nesting+1)
		w.visitAll(node.Else, nesting+1)
	case KindSwitch:
		w.add(node.Label, node.Line, nesting)
		w.visitAll(node.Condition, nesting)
		for _, clause := range node.Handlers {
			w.visitAll(clause.Condition, nesting+1)
			w.visitAll(clause.Body, nesting+1)
			w.visitAll(clause.Children, nesting+1)
		}
		w.visitAll(node.Body, nesting+1)
	case KindLoop:
		if node.Comprehension {
			w.transparent(node, nesting)
			return
		}
		w.add(node.Label, node.Line, nesting)
		w.visitAll(node.Condition, nesting)
		w.visitAll(node.Body, nesting+1)
		if len(node.Else) > 0 || node.HasElse {
			w.add("LoopElse", node.ElseLine, 0)
			w.visitAll(node.Else, nesting+1)
		}
	case KindTry:
		w.visitAll(node.Body, nesting)
		for _, handler := range node.Handlers {
			w.visitCatch(handler, nesting)
		}
		w.visitAll(node.Else, nesting)
		w.visitAll(node.Finally, nesting)
	case KindCatch:
		w.visitCatch(node, nesting)
	case KindJump:
		if node.Labeled {
			w.add(node.Label, node.Line, 0)
		}
	case KindLogical:
		previous := ""
		for _, operator := range node.Operators {
			if !operator.Sequence {
				continue
			}
			if operator.Text != previous {
				w.add("LogicalSequence("+operator.Text+")", operator.Line, 0)
			}
			previous = operator.Text
		}
		w.visitAll(node.Children, nesting)
	default:
		w.transparent(node, nesting)
	}
}

func (w *cognitiveWalker) transparent(node *Node, nesting int) {
	node.each(func(child *Node) { w.visit(child, nesting) })
}

func (w *cognitiveWalker) visitCatch(node *Node, nesting int) {
	w.add(node.Label, node.Line, nesting)
	w.visitAll(node.Condition, nesting)
	w.visitAll(node.Body, nesting+1)
	w.visitAll(node.Children, nesting+1)
}

// visitIf handles if/else-if/else chains: else-if and else get a flat +1, and only
// the first if and plain else blocks raise the nesting of their contents.
func (w *cognitiveWalker) visitIf(node *Node, nesting int, elseIf bool) {
	if elseIf {
		w.add("ElseIf", node.Line, 0)
	} else {
		w.add(node.Label, node.Line, nesting)
	}
	w.visitAll(node.Condition, nesting)
	w.visitAll(node.Body, nesting+1)
	switch {
	case len(node.Else) == 1 && node.Else[0].Kind == KindIf && node.Else[0].ElseIf:
		w.visitIf(node.Else[0], nesting, true)
	case len(node.Else) > 0 || node.HasElse:
		w.add("Else", node.ElseLine, 0)
		w.visitAll(node.Else, nesting+1)
	}
}
