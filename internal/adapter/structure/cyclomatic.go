package structure

// Decision is one counted decision point, recorded as evidence.
type Decision struct {
	Type string `json:"type"`
	Line int    `json:"line"`
}

// CyclomaticResult is the McCabe-family value with its decision points.
type CyclomaticResult struct {
	Value     int
	Decisions []Decision
}

// Cyclomatic is 1 plus every if, else-if, loop, catch, non-default case, ternary,
// comprehension clause and logical operator, excluding nested functions.
func Cyclomatic(function *Function) CyclomaticResult {
	result := CyclomaticResult{Value: 1, Decisions: []Decision{}}
	var visit func(node *Node)
	visit = func(node *Node) {
		if node.Kind == KindFunction {
			return
		}
		switch node.Kind {
		case KindIf, KindLoop, KindCatch, KindTernary:
			result.count(node.Label, node.Line)
		case KindCase:
			if !node.Default {
				result.count(node.Label, node.Line)
			}
		case KindLogical:
			for _, operator := range node.Operators {
				result.count(node.Label+"("+operator.Text+")", operator.Line)
			}
		}
		node.each(visit)
	}
	visit(function.Body)
	return result
}

func (r *CyclomaticResult) count(label string, line int) {
	r.Value++
	r.Decisions = append(r.Decisions, Decision{Type: label, Line: line})
}
