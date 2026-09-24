package cfamily

import (
	"github.com/lumioguard/lumioguard-cc/internal/adapter/cfamily/syntax"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/structure"
)

// lambdaSpecifiers may follow a lambda's parameter list.
var lambdaSpecifiers = set("mutable", "constexpr", "consteval", "static", "noexcept", "throw",
	"__attribute__", "__attribute", "__declspec")

// lambdaParts matches "[captures] <params>? (params)? specifiers {body}" at i
// and returns the parameter list's "(" (or -1) and the body's closing brace.
func (p *parser) lambdaParts(i, to int) (params, closing int, ok bool) {
	if !p.is(i, "[") || p.is(i+1, "[") {
		return -1, 0, false
	}
	j := p.match[i] + 1
	params = -1
	if p.is(j, "<") {
		j = p.skipAngles(j)
	}
	if p.is(j, "(") {
		params, j = j, p.match[j]+1
	}
	if body, ok := p.lambdaBody(j, to); ok {
		return params, p.match[body], true
	}
	return -1, 0, false
}

// lambdaBody steps over a lambda's specifiers and trailing return type from j
// and returns the body's opening brace.
func (p *parser) lambdaBody(j, to int) (int, bool) {
	for j < to {
		switch {
		case p.is(j, "{"):
			return j, true
		case p.is(j, "->") || p.is(j, "requires"):
			j = p.skipToBrace(j+1, to)
		case p.is(j, "["):
			j = p.match[j] + 1
		case p.isWord(j, lambdaSpecifiers):
			j++
			if p.is(j, "(") {
				j = p.match[j] + 1
			}
		default:
			return 0, false
		}
	}
	return 0, false
}

// skipToBrace returns the next top-level "{" in [j, to), or to when a ";"
// comes first.
func (p *parser) skipToBrace(j, to int) int {
	for j < to && !p.is(j, "{") {
		if p.is(j, ";") {
			return to
		}
		j = p.next(j)
	}
	return j
}

// lambda records the lambda at i as a function and returns its node for the
// enclosing function's tree.
func (p *parser) lambda(i, to int, name string, where scope) *structure.Node {
	params, closing, _ := p.lambdaParts(i, to)
	if name == "" {
		name = "<lambda>"
	}
	function := &structure.Function{
		Name:          name,
		ClassName:     where.className,
		EnclosingName: where.function,
		Line:          p.tokens[i].Line,
		Start:         p.tokens[i].Start,
	}
	if params >= 0 {
		function.Parameters = p.parameters(params)
	}
	body := p.block(p.match[closing], scope{className: where.className, function: name})
	p.finish(function, body, closing)
	return &structure.Node{Kind: structure.KindFunction, Label: "LambdaExpression", Line: function.Line, Body: body}
}

// parameters counts the parameter slots of the list opening at open. "(void)"
// declares none; "..." is one slot; commas inside template arguments do not count.
func (p *parser) parameters(open int) int {
	closing := p.match[open]
	if open+1 == closing || closing == open+2 && p.is(open+1, "void") {
		return 0
	}
	count, angles := 1, 0
	for i := open + 1; i < closing; i = p.next(i) {
		if angles == 0 && p.is(i, ",") {
			count++
		}
		angles = p.angleDepth(i, angles)
	}
	return count
}

// angleDepth tracks template argument brackets: "<" opens one only after a
// name, as in "map<int, int>", so "a < b" in a default argument does not.
func (p *parser) angleDepth(i, angles int) int {
	switch {
	case p.is(i, "<") && p.tokens[i-1].Kind == syntax.Identifier:
		return angles + 1
	case p.is(i, ">"):
		return max(angles-1, 0)
	case p.is(i, ">>"):
		return max(angles-2, 0)
	}
	return angles
}
