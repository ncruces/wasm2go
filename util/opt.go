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

// RemoveUnusedLocals replaces unused local variables with the black identifier.
func RemoveUnusedLocals(n ast.Node) {
	// Count identifier uses.
	uses := make(map[string]int)
	ast.Inspect(n, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			uses[id.Name]++
			return false
		}
		return true
	})

	blank := ast.NewIdent("_")

	// If an identifer only shows up once,
	// in the left side of an definition,
	// replace it with the blank identifier.
	// If no names are being defined,
	// turn the definition into an assignment.
	astutil.Apply(n,
		func(c *astutil.Cursor) bool {
			if n, ok := c.Node().(*ast.AssignStmt); ok && n.Tok == token.DEFINE {
				var anydefs bool
				for i := range n.Lhs {
					if id, ok := n.Lhs[i].(*ast.Ident); ok {
						if uses[id.Name] == 1 {
							n.Lhs[i] = blank
						} else if id.Name != blank.Name {
							anydefs = true
						}
					}
				}
				if !anydefs {
					n.Tok = token.ASSIGN
				}
				return false
			}
			return true
		}, nil)
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

// RemoveSelfAssign removes self assignments to locals.
func RemoveSelfAssign(n ast.Node) {
	astutil.Apply(n, func(c *astutil.Cursor) bool {
		if n, ok := c.Node().(*ast.AssignStmt); ok && n.Tok == token.ASSIGN {
			var cloned bool
			for i := len(n.Lhs) - 1; i >= 0; i-- {
				if idL, ok := n.Lhs[i].(*ast.Ident); ok {
					if idR, ok := n.Rhs[i].(*ast.Ident); ok && idL.Name == idR.Name {
						if !cloned {
							cloned = true
							n.Lhs = slices.Clone(n.Lhs)
							n.Rhs = slices.Clone(n.Rhs)
						}
						n.Lhs = slices.Delete(n.Lhs, i, i+1)
						n.Rhs = slices.Delete(n.Rhs, i, i+1)
					}
				}
			}
			if len(n.Lhs) == 0 {
				c.Delete()
			}
			return false
		}
		return true
	}, nil)
}

// UnnestBlocks removes block statements that do not contain variable declarations.
func UnnestBlocks(n ast.Node) {
	astutil.Apply(n, nil, func(c *astutil.Cursor) bool {
		if blk, ok := c.Node().(*ast.BlockStmt); ok {
			// Only unnest if the block is part of a statement list.
			if c.Index() < 0 {
				return true
			}

			var anyvars bool
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
					anyvars = true
					break
				}
				if assign, ok := s.(*ast.AssignStmt); ok && assign.Tok == token.DEFINE {
					anyvars = true
					break
				}
			}

			if !anyvars {
				for _, stmt := range blk.List {
					c.InsertBefore(stmt)
				}
				c.Delete()
			}
		}
		return true
	})
}
