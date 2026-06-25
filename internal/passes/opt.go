package passes

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

// RemoveUnusedLocals replaces unused local variables with the blank identifier.
func RemoveUnusedLocals(fn *ast.FuncDecl) {
	uses := countUses(fn)
	blank := ast.NewIdent("_")

	// If an identifer only shows up once,
	// in the left side of a definition,
	// replace it with the blank identifier.
	// If no names are being defined,
	// turn the definition into an assignment.
	astutil.Apply(fn, func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.ValueSpec:
			for i, id := range n.Names {
				if uses[id.Name] == 1 {
					n.Names[i] = blank
				}
			}
		case *ast.AssignStmt:
			if n.Tok != token.DEFINE {
				break
			}
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
func RemoveSelfAssigns(fn *ast.FuncDecl) {
	uses := countUses(fn)
	writes := countWrites(fn)

	astutil.Apply(fn, func(c *astutil.Cursor) bool {
		if n, ok := c.Node().(*ast.AssignStmt); ok && n.Tok == token.ASSIGN {
			// Skip multi-value assignments of a call.
			if len(n.Lhs) != len(n.Rhs) {
				return true
			}

			var lhs, rhs []ast.Expr
			for i, expr := range n.Rhs {
				// Skip self assignments.
				if idL, ok := n.Lhs[i].(*ast.Ident); ok {
					if idR, ok := expr.(*ast.Ident); ok && idL.Name == idR.Name {
						u := uses[idL.Name] - 2
						w := writes[idL.Name] - 1
						// Don't remove the last read.
						if u > w {
							uses[idL.Name] = u
							writes[idL.Name] = w
							continue
						}
					}
				}
				lhs = append(lhs, n.Lhs[i])
				rhs = append(rhs, expr)
			}

			simplifyAssign(c, n, lhs, rhs)
		}
		return true
	}, nil)
}

// RemoveBlankAssigns removes redundant blank assignments from a variable
// when the variable is read elsewhere.
func RemoveBlankAssigns(fn *ast.FuncDecl) {
	uses := countUses(fn)
	writes := countWrites(fn)

	astutil.Apply(fn, nil, func(c *astutil.Cursor) bool {
		if n, ok := c.Node().(*ast.AssignStmt); ok && n.Tok == token.ASSIGN {
			// Skip multi-value assignments of a call.
			if len(n.Lhs) != len(n.Rhs) {
				return true
			}
			// Skip non-blank assignments.
			for _, expr := range n.Lhs {
				if id, ok := expr.(*ast.Ident); !ok || id.Name != "_" {
					return true
				}
			}

			var lhs, rhs []ast.Expr
			for i, expr := range n.Rhs {
				if id, ok := expr.(*ast.Ident); ok {
					if writes := writes[id.Name]; writes > 0 {
						reads := uses[id.Name] - writes
						if reads > 1 || reads == 1 && writes == 1 {
							uses[id.Name]--
							continue
						}
					}
				}
				lhs = append(lhs, n.Lhs[i])
				rhs = append(rhs, expr)
			}

			simplifyAssign(c, n, lhs, rhs)
		}
		return true
	})
}

// RemoveEmptyStmts removes empty statements preceded by labels.
func RemoveEmptyStmts(fn *ast.FuncDecl) {
	postApplyStmts(fn, func(stmts []ast.Stmt) ([]ast.Stmt, bool) {
		if len(stmts) == 0 {
			return nil, true
		}
		// Iterate backwards so once we find a label with an empty statement
		// we can attach it to the next statement, if it's not a declaration.
		next := len(stmts) - 1
		for i := len(stmts) - 2; i >= 0; i-- {
			stmt := stmts[i]
			if ls, ok := stmt.(*ast.LabeledStmt); ok && is[*ast.EmptyStmt](ls.Stmt) {
				nextStmt := stmts[next]
				if !is[*ast.DeclStmt](nextStmt) {
					ls.Stmt = nextStmt
					stmts[next] = ls
					// If the next statement was already labeled,
					// merge the two labels into one.
					// This works because branches share the
					// identifer instance with the label.
					if inner, ok := ls.Stmt.(*ast.LabeledStmt); ok {
						ls.Label.Name = inner.Label.Name
						stmts[next] = inner
					}
					continue
				}
			}
			next--
			stmts[next] = stmt
		}
		return stmts[next:], true
	})
}

// UnnestSimple removes block statements that
// are the only statement in a statement list.
func UnnestSimple(fn *ast.FuncDecl) {
	postApplyStmts(fn, func(s []ast.Stmt) []ast.Stmt {
		if len(s) == 1 {
			if b, ok := s[0].(*ast.BlockStmt); ok {
				return b.List
			}
		}
		return s
	})
}

// UnnestBlocks removes block statements that
// do not contain variable declarations.
func UnnestBlocks(fn *ast.FuncDecl) {
	astutil.Apply(fn, nil, func(c *astutil.Cursor) bool {
		// A block inside another block.
		if b, ok := c.Node().(*ast.BlockStmt); ok && len(b.List) == 1 {
			if s, ok := b.List[0].(*ast.BlockStmt); ok {
				c.Replace(s)
			}
		}
		return true
	})

	astutil.Apply(fn, nil, func(c *astutil.Cursor) bool {
		// Only unnest if the node is part of a statement list.
		if c.Index() < 0 {
			return true
		}

		var blk *ast.BlockStmt
		var lbl *ast.LabeledStmt
		// Find blocks, and blocks preceded by labels.
		n := c.Node()
		if l, ok := n.(*ast.LabeledStmt); ok {
			n = l.Stmt
			lbl = l
		}
		if b, ok := n.(*ast.BlockStmt); ok {
			blk = b
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
			if is[*ast.DeclStmt](s) {
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
			for i := len(blk.List) - 1; i > 0; i-- {
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
	if is[*ast.ReturnStmt](fn.Body.List[last]) {
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
	astutil.Apply(fn, nil, func(c *astutil.Cursor) bool {
		if branch, ok := c.Node().(*ast.BranchStmt); ok &&
			branch.Tok == token.GOTO && branch.Label.Name == ls.Label.Name {
			c.Replace(&ast.ReturnStmt{})
		}
		return true
	})
}

// InlineGotoReturn replaces a goto to a label with a naked return with a direct return.
func InlineGotoReturn(fn *ast.FuncDecl) {
	found := set[string]{}

	// Find all labels that point directly to a naked return and remove the label.
	astutil.Apply(fn, nil, func(c *astutil.Cursor) bool {
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
	astutil.Apply(fn, nil, func(c *astutil.Cursor) bool {
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
