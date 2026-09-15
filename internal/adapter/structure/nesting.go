package structure

// NestingDepth returns the deepest control-flow nesting in a function body. An
// else-if nests inside its if; nested functions and comprehensions are excluded.
func NestingDepth(function *Function) int {
	maximum := 0
	var descend func(node *Node, depth int)
	descend = func(node *Node, depth int) {
		if node.Kind == KindFunction {
			return
		}
		if node.nests() {
			depth++
		}
		maximum = max(maximum, depth)
		node.each(func(child *Node) { descend(child, depth) })
	}
	descend(function.Body, 0)
	return maximum
}
