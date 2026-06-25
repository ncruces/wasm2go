package passes

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

// Traverses a tree recursively, applying fn in post-order
// to all statement lists.
func postApplyStmts(n ast.Node, fn func([]ast.Stmt) ([]ast.Stmt, bool)) {
	astutil.Apply(n, nil, func(c *astutil.Cursor) (cont bool) {
		switch node := c.Node().(type) {
		case *ast.BlockStmt:
			node.List, cont = fn(node.List)
		case *ast.CaseClause:
			node.Body, cont = fn(node.Body)
		case *ast.CommClause:
			node.Body, cont = fn(node.Body)
		default:
			return true
		}
		return cont
	})
}

// Counts uses of each identifier.
func countUses(n ast.Node) map[string]int {
	uses := map[string]int{}
	unknown := set[string]{}
	ast.Inspect(n, func(n ast.Node) bool {
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

// Counts writes to each identifier.
func countWrites(n ast.Node) map[string]int {
	writes := map[string]int{}
	ast.Inspect(n, func(node ast.Node) bool {
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

// Counts branches to each label.
func countBranches(n ast.Node) map[string]int {
	branches := map[string]int{}
	ast.Inspect(n, func(n ast.Node) bool {
		if br, ok := n.(*ast.BranchStmt); ok && br.Label != nil {
			branches[br.Label.Name]++
		}
		return true
	})
	return branches
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

// Checks if any unlabeled branch escapes n.
func hasEscapingBranch(n ast.Node) bool {
	return canBreak(n) || canContinue(n) || canFallthrough(n)
}

// Checks if an unlabeled break escapes n.
func canBreak(n ast.Node) (found bool) {
	ast.Inspect(n, func(n ast.Node) bool {
		switch n := n.(type) {
		// These reset the scope for unlabeled breaks.
		case *ast.ForStmt, *ast.RangeStmt, *ast.SelectStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.FuncLit:
			return false
		case *ast.BranchStmt:
			if n.Tok == token.BREAK && n.Label == nil {
				found = true
			}
		}
		return !found
	})
	return found
}

// Checks if an unlabeled continue escapes n.
func canContinue(n ast.Node) (found bool) {
	ast.Inspect(n, func(n ast.Node) bool {
		switch n := n.(type) {
		// These reset the scope for unlabeled continues.
		case *ast.ForStmt, *ast.RangeStmt, *ast.FuncLit:
			return false
		case *ast.BranchStmt:
			if n.Tok == token.CONTINUE && n.Label == nil {
				found = true
			}
		}
		return !found
	})
	return found
}

// Checks if a fallthrough escapes n.
func canFallthrough(n ast.Node) (found bool) {
	ast.Inspect(n, func(n ast.Node) bool {
		switch n := n.(type) {
		// These reset the scope for fallthroughs.
		case *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.FuncLit:
			return false
		case *ast.BranchStmt:
			if n.Tok == token.FALLTHROUGH {
				found = true
			}
		}
		return !found
	})
	return found
}

// Checks if s can complete normally.
// It's acceptable to always return true.
func canComplete(s ast.Stmt) bool {
	switch s := s.(type) {
	case *ast.ReturnStmt, *ast.BranchStmt:
		return false
	case *ast.LabeledStmt:
		return canComplete(s.Stmt)
	case *ast.BlockStmt:
		return len(s.List) == 0 || canComplete(s.List[len(s.List)-1])
	case *ast.IfStmt:
		return s.Else == nil || canComplete(s.Body) || canComplete(s.Else)
	case *ast.ExprStmt:
		ce, ok := s.X.(*ast.CallExpr)
		return !ok || callMayReturn(ce)
	case *ast.SwitchStmt:
		var hasDefault bool
		for _, c := range s.Body.List {
			cc := c.(*ast.CaseClause)
			if len(cc.Body) == 0 || canBreak(cc) || canComplete(cc.Body[len(cc.Body)-1]) {
				return true
			}
			if cc.List == nil {
				hasDefault = true
			}
		}
		return !hasDefault
	}
	return true
}

// Checks if an call can return.
func callMayReturn(call *ast.CallExpr) bool {
	id, ok := call.Fun.(*ast.Ident)
	return !ok || id.Name != "panic"
}

// is builtin: http://go.dev/issue/65846
func is[T any](n any) bool {
	_, ok := n.(T)
	return ok
}
