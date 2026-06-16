package passes

import (
	"go/ast"

	"golang.org/x/tools/go/ast/astutil"
)

// Traverses a tree recursively, applying fn in post-order
// to all statement lists.
func postApplyStmts(n ast.Node, fn func([]ast.Stmt) []ast.Stmt) {
	astutil.Apply(n, nil, func(c *astutil.Cursor) bool {
		switch node := c.Node().(type) {
		case *ast.BlockStmt:
			node.List = fn(node.List)
		case *ast.CaseClause:
			node.Body = fn(node.Body)
		case *ast.CommClause:
			node.Body = fn(node.Body)
		}
		return true
	})
}

// Counts uses of an identifier.
func countUses(fn *ast.FuncDecl) map[string]int {
	uses := map[string]int{}
	unknown := set[string]{}
	ast.Inspect(fn, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			uses[n.Name]++
		case *ast.SelectorExpr:
			uses[n.Sel.Name]--
		case *ast.KeyValueExpr:
			if id, ok := n.Key.(*ast.Ident); ok {
				unknown.add(id.Name)
			}
		}
		return true
	})
	for n := range unknown {
		delete(uses, n)
	}
	return uses
}

// Counts writes to an identifier.
func countWrites(fn *ast.FuncDecl) map[string]int {
	writes := map[string]int{}
	ast.Inspect(fn, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.ValueSpec:
			for _, id := range n.Names {
				writes[id.Name]++
			}
		case *ast.AssignStmt:
			for i := range n.Lhs {
				if id, ok := n.Lhs[i].(*ast.Ident); ok {
					writes[id.Name]++
				}
			}
		case *ast.IncDecStmt:
			if id, ok := n.X.(*ast.Ident); ok {
				writes[id.Name]++
			}
		}
		return true
	})
	return writes
}

// Replaces or deletes assignements with a simpler version,
// i.e. removing some or all variables.
func simplifyAssign(c *astutil.Cursor, n *ast.AssignStmt, lhs, rhs []ast.Expr) {
	if len(lhs) == 0 {
		if c.Index() < 0 {
			c.Replace(&ast.EmptyStmt{})
		} else {
			c.Delete()
		}
	} else if len(lhs) < len(n.Lhs) {
		n.Lhs = lhs
		n.Rhs = rhs
	}
}

// http://go.dev/issue/65846
func is[T any](n any) bool {
	_, ok := n.(T)
	return ok
}
