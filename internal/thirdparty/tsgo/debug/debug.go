// Code copied from github.com/microsoft/typescript-go@v0.0.0-20260820064610-89d5d5b2849a.
// Upstream licence: Apache-2.0 (see tsgo/LICENSE and NOTICE.txt).
// Modified by tools/tsgo-sync:
//   - Import paths rewritten from github.com/microsoft/typescript-go/internal/ to this module, because Go forbids importing another module's internal packages.
//   - The experimental github.com/go-json-experiment/json dependency replaced with the Go standard library encoding/json/v2 and encoding/json/jsontext, which is where that experiment was upstreamed.
//   - Test files omitted.

package debug

import (
	"fmt"
)

func Fail(reason string) {
	if len(reason) == 0 {
		reason = "Debug failure."
	} else {
		reason = "Debug failure. " + reason
	}
	// runtime.Breakpoint()
	panic(reason)
}

func FailBadSyntaxKind(node interface{ KindString() string }, message ...any) {
	var msg string
	if len(message) == 0 {
		msg = "Unexpected node."
	} else {
		msg = fmt.Sprint(message...)
	}
	Fail(fmt.Sprintf("%s\nNode %s was unexpected.", msg, node.KindString()))
}

func AssertNever(member any, message ...any) {
	var msg string
	if len(message) == 0 {
		msg = "Illegal value:"
	} else {
		msg = fmt.Sprint(message...)
	}
	var detail string
	if m, ok := member.(interface{ KindString() string }); ok {
		detail = m.KindString()
	} else if m, ok := member.(fmt.Stringer); ok {
		detail = m.String()
	} else {
		detail = fmt.Sprintf("%v", member)
	}
	Fail(fmt.Sprintf("%s %s", msg, detail))
}

func Assert(value bool, message ...any) {
	if value {
		return
	}
	assertSlow(message...)
}

func assertSlow(message ...any) {
	// See https://dave.cheney.net/2020/05/02/mid-stack-inlining-in-go
	var msg string
	if len(message) > 0 {
		msg = "False expression: " + fmt.Sprint(message...)
	} else {
		msg = "False expression."
	}
	Fail(msg)
}
