package syntax

// conditional is one open #if group. Exactly one branch of each group is
// analyzed: the first, or the next one when the first is "#if 0".
type conditional struct {
	name         string
	line         int
	parentActive bool
	active       bool
	taken        bool
	sawElse      bool
}

func (lx *lexer) active() bool {
	return len(lx.conditionals) == 0 || lx.conditionals[len(lx.conditionals)-1].active
}

// conditional applies a conditional directive and reports whether it was one.
func (lx *lexer) conditional(directive *Directive) bool {
	switch directive.Name {
	case "if", "ifdef", "ifndef":
		parent := lx.active()
		keep := !isFalse(directive)
		lx.conditionals = append(lx.conditionals, conditional{
			name: directive.Name, line: directive.Line, parentActive: parent, active: parent && keep, taken: keep,
		})
	case "elif", "elifdef", "elifndef":
		group := lx.open(directive)
		if group.sawElse {
			lx.fail(directive.Line, "#%s after #else", directive.Name)
		}
		keep := !group.taken && !isFalse(directive)
		group.active = group.parentActive && keep
		group.taken = group.taken || keep
	case "else":
		group := lx.open(directive)
		if group.sawElse {
			lx.fail(directive.Line, "#else after #else")
		}
		group.active = group.parentActive && !group.taken
		group.taken, group.sawElse = true, true
	case "endif":
		lx.open(directive)
		lx.conditionals = lx.conditionals[:len(lx.conditionals)-1]
	default:
		return false
	}
	return true
}

// open returns the innermost open group, failing on a stray directive.
func (lx *lexer) open(directive *Directive) *conditional {
	if len(lx.conditionals) == 0 {
		lx.fail(directive.Line, "#%s without #if", directive.Name)
	}
	return &lx.conditionals[len(lx.conditionals)-1]
}

// isFalse reports whether the condition is the literal 0, the usual way to
// disable code. Every other condition is treated as true.
func isFalse(directive *Directive) bool {
	return (directive.Name == "if" || directive.Name == "elif") &&
		len(directive.Tokens) == 3 && directive.Tokens[2].Text == "0"
}
