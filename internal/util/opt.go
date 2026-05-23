package util

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// CheckMaterialized verifies that materialized constants
// (i.e. locals starting with a t) are never reassigned,
// only defined.
func CheckMaterialized(n ast.Node) {
	ast.Inspect(n, func(n ast.Node) bool {
		if assign, ok := n.(*ast.AssignStmt); ok && assign.Tok == token.ASSIGN {
			for _, lhs := range assign.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && strings.HasPrefix(id.Name, "t") {
					panic("assignment to materialized constant: " + id.Name)
				}
			}
		}
		return true
	})
}

// RemoveUnusedLocals replaces unused local variables with the blank identifier.
func RemoveUnusedLocals(n ast.Node) {
	uses := countUses(n)
	blank := ast.NewIdent("_")

	// If an identifer only shows up once,
	// in the left side of an definition,
	// replace it with the blank identifier.
	// If no names are being defined,
	// turn the definition into an assignment.
	astutil.Apply(n,
		func(c *astutil.Cursor) bool {
			if n, ok := c.Node().(*ast.AssignStmt); ok && n.Tok == token.DEFINE {
				var any bool
				for i := range n.Lhs {
					if id, ok := n.Lhs[i].(*ast.Ident); ok {
						if uses[id.Name] == 1 {
							n.Lhs[i] = blank
						} else if id.Name != blank.Name {
							any = true
						}
					}
				}
				if !any {
					n.Tok = token.ASSIGN
				}
			}
			return true
		}, nil)
}

// RemoveSelfAssigns removes self assignments to variables.
func RemoveSelfAssigns(n ast.Node) {
	astutil.Apply(n, func(c *astutil.Cursor) bool {
		if n, ok := c.Node().(*ast.AssignStmt); ok && n.Tok == token.ASSIGN {
			var lhs, rhs []ast.Expr
			for i, expr := range n.Rhs {
				if idL, ok := n.Lhs[i].(*ast.Ident); ok {
					if idR, ok := expr.(*ast.Ident); ok && idL.Name == idR.Name {
						continue
					}
				}
				lhs = append(lhs, n.Lhs[i])
				rhs = append(rhs, expr)
			}

			if len(lhs) == 0 {
				if c.Index() < 0 {
					// cannot delete node in non-list field (e.g., LabeledStmt.Stmt)
					c.Replace(&ast.EmptyStmt{})
				} else {
					c.Delete()
				}
			} else if len(lhs) < len(n.Lhs) {
				n.Lhs = lhs
				n.Rhs = rhs
			}
		}
		return true
	}, nil)
}

// RemoveBlankAssigns removes redundant blank assignments from a variable
// when the variable is read elsewhere.
func RemoveBlankAssigns(n ast.Node) {
	uses := countUses(n)
	writes := countWrites(n)

	astutil.Apply(n, nil, func(c *astutil.Cursor) bool {
		if n, ok := c.Node().(*ast.AssignStmt); ok && n.Tok == token.ASSIGN {
			for _, expr := range n.Lhs {
				if id, ok := expr.(*ast.Ident); !ok || id.Name != "_" {
					return true
				}
			}

			var lhs, rhs []ast.Expr
			for i, expr := range n.Rhs {
				if id, ok := expr.(*ast.Ident); ok {
					if uses[id.Name]-writes[id.Name] > 1 {
						uses[id.Name]--
						continue
					}
				}
				lhs = append(lhs, n.Lhs[i])
				rhs = append(rhs, expr)
			}

			if len(lhs) == 0 {
				c.Delete()
			} else if len(lhs) < len(n.Lhs) {
				n.Lhs = lhs
				n.Rhs = rhs
			}
		}
		return true
	})
}

// RemoveEmptyStmts removes empty statements preceded by labels.
func RemoveEmptyStmts(n ast.Node) {
	astutil.Apply(n, nil,
		func(c *astutil.Cursor) bool {
			// Iterate backwards so once we find a label with an empty statement
			// we can attach it to the next statement, if it's not a declaration.
			if block, ok := c.Node().(*ast.BlockStmt); ok && len(block.List) > 0 {
				stmts := make([]ast.Stmt, 0, len(block.List))
				for i := len(block.List) - 1; i >= 0; i-- {
					stmt := block.List[i]
					if ls, ok := stmt.(*ast.LabeledStmt); ok {
						if _, ok := ls.Stmt.(*ast.EmptyStmt); ok && len(stmts) > 0 {
							nextStmt := stmts[len(stmts)-1]
							if _, ok := nextStmt.(*ast.DeclStmt); !ok {
								ls.Stmt = nextStmt
								stmts[len(stmts)-1] = ls
								// If the next statement was already labeled,
								// merge the two labels into one.
								if inner, ok := ls.Stmt.(*ast.LabeledStmt); ok {
									ls.Label.Name = inner.Label.Name
									stmts[len(stmts)-1] = inner
								}
								continue
							}
						}
					}
					stmts = append(stmts, stmt)
				}
				slices.Reverse(stmts)
				block.List = stmts
			}
			return true
		})
}

// UnnestBlocks removes block statements that do not contain variable declarations.
func UnnestBlocks(n ast.Node) {
	astutil.Apply(n, nil, func(c *astutil.Cursor) bool {
		// Only unnest if the node is part of a statement list.
		if c.Index() < 0 {
			return true
		}

		var blk *ast.BlockStmt
		var lbl *ast.LabeledStmt
		// Find blocks, and blocks preceded by labels.
		if b, ok := c.Node().(*ast.BlockStmt); ok {
			blk = b
		} else if l, ok := c.Node().(*ast.LabeledStmt); ok {
			if b, ok := l.Stmt.(*ast.BlockStmt); ok {
				blk = b
				lbl = l
			}
		}
		if blk == nil {
			return true
		}

		for _, s := range blk.List {
			// Unwrap labeled statements to inspect the actual statement.
			for {
				if ls, ok := s.(*ast.LabeledStmt); ok {
					s = ls.Stmt
				} else {
					break
				}
			}
			// Check for declarations.
			if _, ok := s.(*ast.DeclStmt); ok {
				return true
			}
			if assign, ok := s.(*ast.AssignStmt); ok && assign.Tok == token.DEFINE {
				return true
			}
		}

		// Unnest bare blocks.
		if lbl == nil {
			for _, stmt := range blk.List {
				c.InsertBefore(stmt)
			}
			c.Delete()
			return true
		}

		// Unnest labled blocks.
		if len(blk.List) == 0 {
			lbl.Stmt = &ast.EmptyStmt{}
		} else {
			lbl.Stmt = blk.List[0]
			for i := len(blk.List) - 1; i >= 1; i-- {
				c.InsertAfter(blk.List[i])
			}
		}
		return true
	})
}

// InlineGotoEnd replaces a goto to the end of a function with return.
func InlineGotoEnd(fn *ast.FuncDecl) {
	last := len(fn.Body.List) - 1

	// A non-empty function that returns no values.
	if last < 0 || fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		return
	}
	// A return statement at the end.
	if _, ok := fn.Body.List[last].(*ast.ReturnStmt); ok {
		fn.Body.List = fn.Body.List[:last]
		return
	}
	// A labeled statement at the end.
	ls, ok := fn.Body.List[last].(*ast.LabeledStmt)
	if !ok {
		return
	}

	// That's either empty or return.
	switch ls.Stmt.(type) {
	case *ast.EmptyStmt, *ast.ReturnStmt:
		break
	default:
		return
	}

	// Remove the statement.
	fn.Body.List = fn.Body.List[:last]

	// Fix the branches.
	astutil.Apply(fn.Body, nil, func(c *astutil.Cursor) bool {
		if branch, ok := c.Node().(*ast.BranchStmt); ok &&
			branch.Tok == token.GOTO && branch.Label.Name == ls.Label.Name {
			c.Replace(&ast.ReturnStmt{})
		}
		return true
	})
}

// InlineGotoReturn replaces a goto to a label with a naked return with a direct return.
func InlineGotoReturn(fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}

	found := set[string]{}

	// Find all labels that point directly to a naked return and remove the label.
	astutil.Apply(fn.Body, nil, func(c *astutil.Cursor) bool {
		if ls, ok := c.Node().(*ast.LabeledStmt); ok {
			if ret, ok := ls.Stmt.(*ast.ReturnStmt); ok && len(ret.Results) == 0 {
				found.add(ls.Label.Name)
				c.Replace(ret)
			}
		}
		return true
	})

	if len(found) == 0 {
		return
	}

	// Replace gotos to those labels with a naked return.
	ret := &ast.ReturnStmt{}
	astutil.Apply(fn.Body, nil, func(c *astutil.Cursor) bool {
		if branch, ok := c.Node().(*ast.BranchStmt); ok && branch.Tok == token.GOTO {
			if found.has(branch.Label.Name) {
				c.Replace(ret)
			}
		}
		return true
	})
}

// RemoveReceiver converts a method that doesn't use their receiver
// into a plain function.
// Returns true if we need to update call sites.
func RemoveReceiver(fn *ast.FuncDecl) bool {
	if ast.IsExported(fn.Name.Name) {
		return false
	}

	unused := true
	name := fn.Recv.List[0].Names[0].Name
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if unused {
			if id, ok := n.(*ast.Ident); ok && id.Name == name {
				unused = false
			}
		}
		return unused
	})

	if unused {
		fn.Recv = nil
	}
	return unused
}

// RemoveParens removes all ParenExpr nodes.
// These are always unnecessary in generated code.
// The printer/formatter will automatically introduce whatever parens
// are necessary to preserve precedence of operators.
func RemoveParens(n ast.Node) {
	astutil.Apply(n, nil, func(c *astutil.Cursor) bool {
		if paren, ok := c.Node().(*ast.ParenExpr); ok {
			c.Replace(paren.X)
		}
		return true
	})
}

// Counts uses of an identifier.
func countUses(n ast.Node) map[string]int {
	uses := make(map[string]int)
	ast.Inspect(n, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			uses[id.Name]++
		}
		return true
	})
	return uses
}

// Counts writes to an identifier.
func countWrites(n ast.Node) map[string]int {
	writes := make(map[string]int)
	ast.Inspect(n, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.ValueSpec:
			for _, id := range x.Names {
				writes[id.Name]++
			}
		case *ast.AssignStmt:
			for i := range x.Lhs {
				if id, ok := x.Lhs[i].(*ast.Ident); ok {
					writes[id.Name]++
				}
			}
		}
		return true
	})
	return writes
}
