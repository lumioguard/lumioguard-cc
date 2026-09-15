package structure

import "testing"

func fn(body ...*Node) *Function {
	return &Function{Name: "f", Body: &Node{Kind: KindBlock, Children: body}}
}

func ifNode(line int, body ...*Node) *Node {
	return &Node{Kind: KindIf, Label: "If", Line: line, Body: body}
}

func loop(line int, body ...*Node) *Node {
	return &Node{Kind: KindLoop, Label: "Loop", Line: line, Body: body}
}

func logical(text ...string) *Node {
	node := &Node{Kind: KindLogical, Label: "Logical", Line: 1}
	for _, op := range text {
		node.Operators = append(node.Operators, Operator{Text: op, Line: 1, Sequence: op != "??"})
	}
	return node
}

func TestCyclomaticCountsBranchesLoopsCatchesCasesAndOperators(t *testing.T) {
	function := fn(
		ifNode(1),
		loop(2),
		&Node{Kind: KindTry, Handlers: []*Node{{Kind: KindCatch, Label: "Catch", Line: 3}}},
		&Node{Kind: KindSwitch, Handlers: []*Node{{Kind: KindCase, Label: "Case", Line: 4}, {Kind: KindCase, Default: true, Line: 5}}},
		&Node{Kind: KindTernary, Label: "Ternary", Line: 6},
		logical("&&", "||", "??"),
		&Node{Kind: KindFunction, Body: []*Node{ifNode(8)}},
	)
	result := Cyclomatic(function)
	if result.Value != 9 {
		t.Fatalf("cyclomatic = %d, want 9 (%+v)", result.Value, result.Decisions)
	}
	if len(result.Decisions) != 8 {
		t.Fatalf("decisions = %+v", result.Decisions)
	}
}

func TestNestingDepthSkipsNestedFunctionsAndComprehensions(t *testing.T) {
	function := fn(
		ifNode(1, loop(2, &Node{Kind: KindFunction, Body: []*Node{ifNode(3, ifNode(4))}})),
		&Node{Kind: KindLoop, Comprehension: true, Body: []*Node{{Kind: KindIf, Comprehension: true}}},
	)
	if depth := NestingDepth(function); depth != 2 {
		t.Fatalf("nesting = %d, want 2", depth)
	}
}

func TestCognitiveElseIfChainAndElseBlock(t *testing.T) {
	chain := ifNode(1)
	chain.Else = []*Node{{Kind: KindIf, Label: "If", Line: 2, ElseIf: true, Else: []*Node{loop(3)}}}
	chain.Else[0].ElseLine = 3
	// if +1, else-if +1 (flat), else +1 (flat), loop inside else at nesting 1 → +2 = 5
	if value := Cognitive(fn(chain)).Value; value != 5 {
		t.Fatalf("cognitive = %d, want 5", value)
	}
	nested := ifNode(1)
	nested.ElseLine = 2
	nested.Else = []*Node{ifNode(2)} // "else { if }" is not an else-if: else +1, inner if +1+1
	if value := Cognitive(fn(nested)).Value; value != 4 {
		t.Fatalf("cognitive = %d, want 4", value)
	}
	empty := ifNode(1)
	empty.HasElse = true // "if (a) {} else {}" still counts the else branch
	if value := Cognitive(fn(empty)).Value; value != 2 {
		t.Fatalf("cognitive = %d, want 2", value)
	}
}

func TestCognitiveOperatorSequencesAndLabeledJumps(t *testing.T) {
	function := fn(
		logical("&&", "&&", "||", "||", "&&"),
		logical("??", "??"),
		&Node{Kind: KindJump, Label: "Break", Labeled: true, Line: 3},
		&Node{Kind: KindJump, Label: "Break", Labeled: false, Line: 4},
		&Node{Kind: KindLoop, Label: "Loop", Line: 5, Comprehension: true, Body: []*Node{ifNode(6)}},
	)
	// sequences: && (+1), || (+1), && (+1); ?? ignored; labelled jump +1; comprehension transparent, inner if +1
	if value := Cognitive(function).Value; value != 5 {
		t.Fatalf("cognitive = %d, want 5", value)
	}
}

func TestCognitiveNestedFunctionsRaiseNestingAndLoopElse(t *testing.T) {
	function := fn(
		&Node{Kind: KindFunction, Body: []*Node{ifNode(2)}},
		&Node{Kind: KindLoop, Label: "Loop", Line: 3, Else: []*Node{ifNode(4)}, ElseLine: 4},
		&Node{Kind: KindTry, Body: []*Node{ifNode(5)}, Handlers: []*Node{{Kind: KindCatch, Label: "Catch", Line: 6, Body: []*Node{ifNode(7)}}}},
	)
	// nested if at nesting 1 (+2); loop +1; loop else +1; if in else at nesting 1 (+2);
	// if in try body +1; catch +1; if in catch at nesting 1 (+2) = 10
	if value := Cognitive(function).Value; value != 10 {
		t.Fatalf("cognitive = %d, want 10", value)
	}
}

func TestAssignSymbolsOrdersAndDeduplicates(t *testing.T) {
	functions := []*Function{
		{Name: "b", Start: 20, Line: 2, EndLine: 3, Body: &Node{Kind: KindBlock}},
		{Name: "a", ClassName: "C", Start: 5, Line: 1, EndLine: 1, Body: &Node{Kind: KindBlock}},
		{Name: "b", Start: 30, Line: 4, EndLine: 5, Body: &Node{Kind: KindBlock}},
		{Name: "<lambda>", EnclosingName: "b", ClassName: "C", Start: 25, Line: 2, EndLine: 2, Body: &Node{Kind: KindBlock}},
	}
	records := AssignSymbols("x.py", functions)
	want := []string{"C.a", "b", "C.b.<lambda>", "b#2"}
	for i, record := range records {
		if record.Scope.Symbol != want[i] {
			t.Fatalf("symbols = %v, want %v", symbolsOf(records), want)
		}
		if record.Scope.Key != "x.py::"+want[i] {
			t.Fatalf("scope key = %q", record.Scope.Key)
		}
	}
}

func symbolsOf(records []Record) []string {
	result := make([]string, len(records))
	for i, record := range records {
		result[i] = record.Scope.Symbol
	}
	return result
}
