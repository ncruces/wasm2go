package passes

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

// HoistVars lifts wasm2go's per-block temporaries to function scope.
// This unlocks more UnnestBlocks candidates (Go forbids a goto jumping over an
// in-scope declaration) which can flatten deeply-nested block constructs.
//
// Codegen emits temps long-form (`var tN T = e`); HoistVars splits each into a
// hoisted `var tN T` plus an in-place `tN = e`, then UnnestBlocks collapses the
// cascade. Safe because temps are single-assignment and uniquely named; the type
// rides in on the decl (no inference). Runs before UnnestBlocks/InlineSingleGoto.
func HoistVars(fn *ast.FuncDecl) {
	if fn.Body != nil {
		hoistFuncVars(fn)
	}
}

type hoistedDecl struct {
	name string
	typ  ast.Expr
}

func hoistFuncVars(fn *ast.FuncDecl) {
	// Count declaration sites per name across the whole function.
	// A name declared more than once will not be hoisted: sibling blocks may
	// legally declare the same temp name as different types and, to simplify
	// reasoning about safety, we detect and skip these hoists.
	declCount := map[string]int{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.GenDecl: // long form: var x T  /  var x T = e
			if n.Tok == token.VAR {
				for _, spec := range n.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						for _, id := range vs.Names {
							declCount[id.Name]++
						}
					}
				}
			}
		case *ast.AssignStmt: // short form: x := e
			if n.Tok == token.DEFINE {
				for _, lhs := range n.Lhs {
					if id, ok := lhs.(*ast.Ident); ok {
						declCount[id.Name]++
					}
				}
			}
		}
		return true
	})
	hoistable := func(name string) bool { return name != "_" && declCount[name] == 1 }
	var hoisted []hoistedDecl
	astutil.Apply(fn.Body, nil, func(c *astutil.Cursor) bool {
		if ds, ok := c.Node().(*ast.DeclStmt); !ok {
			return true
		} else if gd, ok := ds.Decl.(*ast.GenDecl); !ok || gd.Tok != token.VAR || len(gd.Specs) != 1 {
			return true
		} else if vs, ok := gd.Specs[0].(*ast.ValueSpec); !ok || vs.Type == nil {
			return true
		} else if len(vs.Values) == 0 { // Decl form `var ... T`
			if c.Parent() == fn.Body { // the function's own locals, already at top
				return true
			}
			// Nested: condition temps / block results. Hoist only when every name
			// is uniquely declared; otherwise leave the declaration in place.
			for _, id := range vs.Names {
				if id.Name != "_" && !hoistable(id.Name) {
					return true
				}
			}
			// Hoist the bare `var x T` but leave a `x = 0` value reset.
			var lhs, rhs []ast.Expr
			for _, id := range vs.Names {
				if id.Name == "_" {
					continue
				}
				hoisted = append(hoisted, hoistedDecl{id.Name, vs.Type})
				lhs = append(lhs, id)
				rhs = append(rhs, zeroValue(vs.Type))
			}
			if len(lhs) == 0 {
				c.Delete() // e.g. `var _ T` -- nothing to hoist
			} else {
				c.Replace(&ast.AssignStmt{Tok: token.ASSIGN, Lhs: lhs, Rhs: rhs})
			}
		} else if len(vs.Names) == 1 { // Value form `var x T = e`
			name := vs.Names[0].Name
			switch {
			case name == "_":
				// Blanked by RemoveUnusedLocals: no declaration needed, but keep
				// rhs's side effects and simplify assign so block can flatten.
				c.Replace(valueAssign(vs))
			case hoistable(name):
				// Hoist `var x T` and leave `x = e` in place.
				hoisted = append(hoisted, hoistedDecl{name, vs.Type})
				c.Replace(valueAssign(vs))
			default:
				// Declared more than once: leave the block-scoped declaration.
			}
		}
		return true
	})
	if len(hoisted) == 0 {
		return
	}
	decls := make([]ast.Stmt, len(hoisted))
	for i, h := range hoisted {
		decls[i] = &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{
			&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent(h.name)}, Type: h.typ}}}}
	}
	fn.Body.List = append(decls, fn.Body.List...)
}

// valueAssign turns a `var x T = e` spec into the assignment `x = e`.
func valueAssign(vs *ast.ValueSpec) *ast.AssignStmt {
	return &ast.AssignStmt{Tok: token.ASSIGN, Lhs: []ast.Expr{vs.Names[0]}, Rhs: []ast.Expr{vs.Values[0]}}
}

// zeroValue is the zero value used to re-initialize a hoisted no-initializer
func zeroValue(typ ast.Expr) ast.Expr {
	if id, ok := typ.(*ast.Ident); ok && id.Name == "any" {
		return ast.NewIdent("nil")
	}
	return &ast.BasicLit{Kind: token.INT, Value: "0"}
}
