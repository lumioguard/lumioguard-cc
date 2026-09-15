// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package ast

// FlowFlags

type FlowFlags uint32

const (
	FlowFlagsUnreachable    FlowFlags = 1 << 0  // Unreachable code
	FlowFlagsStart          FlowFlags = 1 << 1  // Start of flow graph
	FlowFlagsBranchLabel    FlowFlags = 1 << 2  // Non-looping junction
	FlowFlagsLoopLabel      FlowFlags = 1 << 3  // Looping junction
	FlowFlagsAssignment     FlowFlags = 1 << 4  // Assignment
	FlowFlagsTrueCondition  FlowFlags = 1 << 5  // Condition known to be true
	FlowFlagsFalseCondition FlowFlags = 1 << 6  // Condition known to be false
	FlowFlagsSwitchClause   FlowFlags = 1 << 7  // Switch statement clause
	FlowFlagsArrayMutation  FlowFlags = 1 << 8  // Potential array mutation
	FlowFlagsCall           FlowFlags = 1 << 9  // Potential assertion call
	FlowFlagsReduceLabel    FlowFlags = 1 << 10 // Temporarily reduce antecedents of label
	FlowFlagsReferenced     FlowFlags = 1 << 11 // Referenced as antecedent once
	FlowFlagsShared         FlowFlags = 1 << 12 // Referenced as antecedent more than once
	FlowFlagsLabel                    = FlowFlagsBranchLabel | FlowFlagsLoopLabel
	FlowFlagsCondition                = FlowFlagsTrueCondition | FlowFlagsFalseCondition
)

// FlowNode

type FlowNode struct {
	Flags       FlowFlags
	Node        *Node     // Associated AST node
	Antecedent  *FlowNode // Antecedent for all but FlowLabel
	Antecedents *FlowList // Linked list of antecedents for FlowLabel
}

type FlowList struct {
	Flow *FlowNode
	Next *FlowList
}

type FlowLabel = FlowNode

// FlowSwitchClauseData (synthetic AST node for FlowFlagsSwitchClause)

type FlowSwitchClauseData struct {
	NodeBase
	SwitchStatement *Node
	ClauseStart     int32 // Start index of case/default clause range
	ClauseEnd       int32 // End index of case/default clause range
}

func NewFlowSwitchClauseData(switchStatement *Node, clauseStart int, clauseEnd int) *Node {
	node := &FlowSwitchClauseData{}
	node.SwitchStatement = switchStatement
	node.ClauseStart = int32(clauseStart)
	node.ClauseEnd = int32(clauseEnd)
	return newNode(KindUnknown, node, NodeFactoryHooks{})
}

func (node *FlowSwitchClauseData) IsEmpty() bool {
	return node.ClauseStart == node.ClauseEnd
}

// FlowReduceLabelData (synthetic AST node for FlowFlagsReduceLabel)

type FlowReduceLabelData struct {
	NodeBase
	Target      *FlowLabel // Target label
	Antecedents *FlowList  // Temporary antecedent list
}

func NewFlowReduceLabelData(target *FlowLabel, antecedents *FlowList) *Node {
	node := &FlowReduceLabelData{}
	node.Target = target
	node.Antecedents = antecedents
	return newNode(KindUnknown, node, NodeFactoryHooks{})
}
