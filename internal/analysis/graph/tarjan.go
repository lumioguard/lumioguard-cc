package graph

import (
	"maps"
	"slices"
)

// stronglyConnectedComponents returns Tarjan components of two or more modules,
// in sorted order. A self-import alone is not a cycle.
func stronglyConnectedComponents(nodes []string, edges map[string]map[string]struct{}) [][]string {
	next := 0
	indices := make(map[string]int, len(nodes))
	lows := make(map[string]int, len(nodes))
	onStack := make(map[string]bool, len(nodes))
	var stack []string
	var components [][]string

	var connect func(node string)
	connect = func(node string) {
		indices[node] = next
		lows[node] = next
		next++
		stack = append(stack, node)
		onStack[node] = true
		for _, target := range slices.Sorted(maps.Keys(edges[node])) {
			if _, visited := indices[target]; !visited {
				connect(target)
				lows[node] = min(lows[node], lows[target])
			} else if onStack[target] {
				lows[node] = min(lows[node], indices[target])
			}
		}
		if lows[node] != indices[node] {
			return
		}
		var component []string
		for {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[top] = false
			component = append(component, top)
			if top == node {
				break
			}
		}
		if len(component) > 1 {
			slices.Sort(component)
			components = append(components, component)
		}
	}

	for _, node := range nodes {
		if _, visited := indices[node]; !visited {
			connect(node)
		}
	}
	slices.SortFunc(components, slices.Compare)
	return components
}
