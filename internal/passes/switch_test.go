package passes

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"testing"
)

func TestInlineSwitchTargets(t *testing.T) {
	for _, tc := range []struct {
		name          string
		input, expect string
	}{{
		name:   "inline all targets",
		input:  `{{switch x { case 0: goto l0; default: goto l1}};l0:;a()};l1:;b()`,
		expect: `switch x { case 0: a(); fallthrough; default: b()}`,
	}, {
		name:   "inline sole default",
		input:  `{switch x { default: goto l0}};l0:;a()`,
		expect: `switch x { default: a()}`,
	}, {
		name:   "multi-entry target ends the run",
		input:  `{{switch x { case 0: goto l0; default: goto l1}};l0:;a()};l1:;b();if c() {goto l1}`,
		expect: `switch x { default: goto l1; case 0: a()};l1:;b();if c() {goto l1}`,
	}, {
		name:   "terminating segment needs no fallthrough",
		input:  `{{switch x { case 0: goto l0; default: goto l1}};l0:;return};l1:;b()`,
		expect: `switch x { case 0: return; default: b()}`,
	}, {
		name:   "panic terminates a segment",
		input:  `{{switch x { case 0: goto l0; default: goto l1}};l0:;panic("x")};l1:;b()`,
		expect: `switch x { case 0: panic("x"); default: b()}`,
	}, {
		name:   "trailing goto terminates a segment",
		input:  `lx:;{{switch x { case 0: goto l0; default: goto l1}};l0:;a();goto lx};l1:;b()`,
		expect: `lx:;switch x { case 0: a();goto lx; default: b()}`,
	}, {
		name:   "chained dispatches reuse the switch",
		input:  `{{{switch x { case 0: goto l1; default: goto l0 }};l0:;a()};l1:;goto l2};l2:;b()`,
		expect: `switch x { default: a(); fallthrough; case 0: goto l2};l2:;b()`,
	}, {
		name:   "fallthrough target stays in place",
		input:  `{{{{switch x { case 0: goto l0; case 1: goto l2; default: goto l1 }};l0:};l2:;goto l3};l1:;panic("x")};l3:;b()`,
		expect: `switch x { case 0: fallthrough; case 1: goto l3; default: panic("x")};l3:;b()`,
	}, {
		name:   "pre-dispatch declarations keep their scope",
		input:  `{{y := p(); switch y { case 0: goto l0; default: goto l1}};l0:;a()};l1:;b()`,
		expect: `{y := p(); switch y { case 0: a(); fallthrough; default: b()}}`,
	}, {
		name:   "declaration-free pre-dispatch code unnests",
		input:  `{{p(); switch x { case 0: goto l0; default: goto l1}};l0:;a()};l1:;b()`,
		expect: `p();switch x { case 0: a(); fallthrough; default: b()}`,
	}, {
		name:   "grouped case values",
		input:  `{{switch x { case 0, 1: goto l0; default: goto l1}};l0:;a()};l1:;b()`,
		expect: `switch x { case 0, 1: a(); fallthrough; default: b()}`,
	}, {
		name:   "label on a statement is not a ladder",
		input:  `{switch x { default: goto l0}};l0:a()`,
		expect: `switch x { default: goto l0};l0:a()`,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got := rewrite(t, tc.input, true)
			want := rewrite(t, tc.expect, false)
			if got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// rewrite parses src and conditionally applies InlineSwitchTargets and UnnestBlocks.
func rewrite(t *testing.T, src string, transform bool) string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", "package p\nfunc f() {\n"+src+"\n}", parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	if transform {
		InlineSwitchTargets(fn)
		UnnestBlocks(fn)
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, fn); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
