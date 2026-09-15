package syntax

import (
	"strings"
	"testing"
)

const modernSource = `#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Module docstring with 'quotes' and \"\"\"nested\"\"\" markers."""
from __future__ import annotations
import os.path as osp, sys
from . import sibling
from ..package.module import (name_one, name_two as alias,)
from typing import *

CONSTANT: int = 0x_FF + 0b1010 + 0o17 + 1_000.5e-3 + 3j
message = f"{CONSTANT!r:>{10}} and {'nested' if CONSTANT else "other"} {{literal}}"
raw = rb'\x00\'' + Rb"\"" + '''multi
line''' + r'''raw \
continued'''
values = [x ** 2 for x in range(10) if x % 2 == 0 if x > 2]
matrix = {k: [v for v in vs if v] for k, vs in items.items()}
unique = {(a, b) for a in xs for b in ys}
generator = (yield_value async for yield_value in aiter(source))
sliced = values[1:2, ::3, ...]
walrus = [y for x in data if (y := f(x)) is not None]
chained = 1 < x <= 10 != y not in z is not None
lam = lambda a, /, b=1, *args, c, d=2, **kw: a if b else c
tern = x if y else z

@decorator(arg=1)
@other.decorator
async def fetch(url: str, *, retries: int = 3, **options: dict[str, int]) -> "Response":
    async with session.get(url) as response, other() as second:
        async for chunk in response:
            await handle(chunk)
    try:
        pass
    except* (ValueError, TypeError) as error:
        raise RuntimeError("failed") from error
    except Exception:
        return None
    else:
        pass
    finally:
        cleanup()
    while True:
        break
    else:
        continue
    return await response.json()

class Point[T](Base, metaclass=Meta):
    x: int = 0

    def __init__(self, x: int, y: int) -> None:
        self.x, self.y = x, y
        global CONSTANT
        nonlocal_free = None
        del nonlocal_free
        assert x >= 0, "negative"

    @property
    def norm(self) -> float:
        return (self.x ** 2 + self.y ** 2) ** 0.5

type Alias[T] = list[T]

def describe(point):
    match point:
        case Point(x=0, y=0):
            return "origin"
        case [1, *rest] | (2, 3) as pair if rest:
            return "sequence"
        case {"key": value, **others}:
            return value
        case Point() | str():
            return "typed"
        case _:
            return "unknown"
    match = 1
    match(point)
    print(match)

with (open("a") as fa, open("b") as fb):
    fa.write(fb.read()); fb.close()
if x: y = 1; z = 2
elif w:
    pass
else: pass
`

func TestParsesModernPythonSyntax(t *testing.T) {
	module, tokens, comments, err := Parse(modernSource)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(tokens) == 0 || len(comments) != 2 {
		t.Fatalf("tokens=%d comments=%d", len(tokens), len(comments))
	}
	var functions []string
	Walk(module, func(node *Node) bool {
		if node.Kind == FunctionDef {
			functions = append(functions, node.Name)
		}
		return true
	})
	want := []string{"fetch", "__init__", "norm", "describe"}
	if strings.Join(functions, ",") != strings.Join(want, ",") {
		t.Fatalf("functions = %v, want %v", functions, want)
	}
}

func TestPositionsAndParameterSlots(t *testing.T) {
	source := "@dec\ndef f(a, b=1, *args, c, **kw):\n    x = (\n        1,\n        2,\n    )\n    return x\n\ndef g(): pass\n"
	module, _, _, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	f := module.Body[0]
	if f.Kind != FunctionDef || f.Name != "f" || f.Params != 5 {
		t.Fatalf("unexpected function %+v", f)
	}
	if f.Line != 2 || f.EndLine != 7 {
		t.Fatalf("f spans %d-%d, want 2-7", f.Line, f.EndLine)
	}
	if got := source[f.Start:f.End]; !strings.HasPrefix(got, "def f(") || !strings.HasSuffix(got, "return x") {
		t.Fatalf("unexpected function text %q", got)
	}
	g := module.Body[1]
	if g.Line != 9 || g.EndLine != 9 || g.Params != 0 {
		t.Fatalf("g spans %d-%d params %d", g.Line, g.EndLine, g.Params)
	}
}

func TestElifIsDistinguishedFromNestedElseIf(t *testing.T) {
	module, _, _, err := Parse("if a:\n    pass\nelif b:\n    pass\nelse:\n    if c:\n        pass\n")
	if err != nil {
		t.Fatal(err)
	}
	outer := module.Body[0]
	if len(outer.OrElse) != 1 || !outer.OrElse[0].Elif || outer.ElseLine != 3 {
		t.Fatalf("expected elif chain, got %+v", outer.OrElse)
	}
	inner := outer.OrElse[0]
	if len(inner.OrElse) != 1 || inner.OrElse[0].Elif || inner.OrElse[0].Kind != If || inner.ElseLine != 5 {
		t.Fatalf("expected else block containing if, got %+v", inner.OrElse)
	}
}

func TestBoolOpChainsRecordOperatorLines(t *testing.T) {
	module, _, _, err := Parse("x = (a and\n     b and c or d)\n")
	if err != nil {
		t.Fatal(err)
	}
	value := module.Body[0].Values[1]
	if value.Kind != BoolOp || value.Op != "or" || len(value.Values) != 2 {
		t.Fatalf("expected or-chain, got %+v", value)
	}
	inner := value.Values[0]
	if inner.Kind != BoolOp || inner.Op != "and" || len(inner.Values) != 3 || len(inner.OpLines) != 2 || inner.OpLines[0] != 1 || inner.OpLines[1] != 2 {
		t.Fatalf("expected and-chain with operator lines, got %+v", inner)
	}
}

func TestSyntaxErrorsReportLine(t *testing.T) {
	cases := map[string]int{
		"def broken(:\n    pass\n":  1,
		"x = 1\nif x\n    pass\n":   2,
		"x = 'unterminated\n":       1,
		"def f():\n  pass\n pass\n": 3,
		"print 'python 2'\n":        1,
		"x = (1,\n":                 2,
		"f = f'{unclosed'\n":        1,
	}
	for source, line := range cases {
		_, _, _, err := Parse(source)
		if err == nil {
			t.Errorf("expected error for %q", source)
			continue
		}
		if syntaxError, ok := err.(*Error); !ok || syntaxError.Line != line {
			t.Errorf("%q: error %v, want line %d", source, err, line)
		}
	}
}

func TestFStringBackslashesAndBraces(t *testing.T) {
	sources := []string{
		"x = fr'\\{{'\n",
		"x = rf'({_name})({_ws}*)(\\{{)'\n",
		"x = fr'\\{{{1+1}'\n",
		"x = f\"\\N{BULLET} {value:>{width}}\"\n",
		"x = f'{\"a\" if b else \"c\"} {{literal}} {d!r}'\n",
		"x = f'''multi {\n    line\n} field'''\n",
	}
	for _, source := range sources {
		module, tokens, _, err := Parse(source)
		if err != nil {
			t.Errorf("%q: %v", source, err)
			continue
		}
		if len(module.Body) != 1 || module.Body[0].Kind != Assign {
			t.Errorf("%q: expected one assignment", source)
		}
		strings := 0
		for _, token := range tokens {
			if token.Type == String {
				strings++
			}
		}
		if strings != 1 {
			t.Errorf("%q: expected one string token, got %d", source, strings)
		}
	}
}

func TestTokenizeStructuralTokensAndStrings(t *testing.T) {
	tokens, comments, err := Tokenize("def f():  # c\n    return f\"{x}\" 'y'\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 {
		t.Fatalf("comments = %+v", comments)
	}
	var kinds []string
	for _, token := range tokens {
		kinds = append(kinds, token.Type.String())
	}
	want := "Name Name Op Op Op Newline Indent Name String String Newline Dedent EOF"
	if got := strings.Join(kinds, " "); got != want {
		t.Fatalf("token kinds = %q, want %q", got, want)
	}
	if tokens[8].Text != "f\"{x}\"" {
		t.Fatalf("f-string token = %q", tokens[8].Text)
	}
}
