package passes

import (
	"go/ast"
	"go/token"
)

// InlineSwitchGotos inlines switch cases that consist of a single goto,
// to an otherwise unused label.
func InlineSwitchGotos(fn *ast.FuncDecl) {
	uses := map[string]int{}
	ast.Inspect(fn, func(n ast.Node) bool {
		if br, ok := n.(*ast.BranchStmt); ok {
			uses[br.Label.Name]++
		}
		return true
	})

	// This loop applies the optimization iteratively,
	// re-verifying conditions after every modification.
	for modified := true; modified; {
		modified = false
		postApplyStmts(fn, func(stmts []ast.Stmt) []ast.Stmt {
			for i := 0; i+1 < len(stmts); i++ {
				// A labeled statement, followed by an empty statement,
				// only used once as a goto target.
				ls, ok := stmts[i+1].(*ast.LabeledStmt)
				if !ok || !is[*ast.EmptyStmt](ls.Stmt) || uses[ls.Label.Name] != 1 {
					continue
				}
				// Find the preceeding switch statement.
				sw := findSwitchStmt(stmts[i])
				if sw == nil {
					continue
				}
				// Find the case label, and validate the optimization.
				id := findSwitchCase(sw, ls.Label.Name)
				if id < 0 {
					continue
				}
				// Perform the optimization, and try again.
				inlineSwitchCase(sw, id, stmts[i+2:])
				stmts = stmts[:i+1]
				modified = true
				break
			}
			return stmts
		})
	}
}

// Finds the switch statement at the end of s.
func findSwitchStmt(s ast.Stmt) *ast.SwitchStmt {
	switch s := s.(type) {
	case *ast.SwitchStmt:
		return s
	case *ast.BlockStmt:
		if len := len(s.List); len > 0 {
			return findSwitchStmt(s.List[len-1])
		}
	}
	return nil
}

// Finds the index of a case clause whose body is goto label;
// returns -1 if inlining it would be invalid.
func findSwitchCase(sw *ast.SwitchStmt, label string) (idx int) {
	// If such a case clause is found,
	// it should move to the end of the switch statement,
	// and the label will be "inlined" into the case clause.
	//
	// To move the clause to the end of the switch statement,
	// the preceeding clause cannot end in a fallthrough statement.
	//
	// To move the label into the switch,
	// execution cannot naturally flow out of the switch statement
	// except through the current final clause,
	// in which case we will need to add a fallthrough to that clause.
	//
	// To ensure execution cannot flow out of the switch statement,
	// a default clause must exist, and all clauses (except the final one)
	// must return/goto/continue/falthrough and never break.
	idx = -1
	var hasDefault bool
	var hadFallthrough bool
	for i, c := range sw.Body.List {
		cc := c.(*ast.CaseClause)
		// Remember if the switch has a default case.
		if cc.List == nil {
			hasDefault = true
		}
		// Check that the case is neither empty nor breaks.
		if len(cc.Body) == 0 || hasBreak(cc) {
			return -1
		}
		switch c := cc.Body[len(cc.Body)-1].(type) {
		case *ast.ReturnStmt:
			hadFallthrough = false
		case *ast.BranchStmt:
			// Is the body just `goto label`?
			// Does it need to move, and can it be moved?
			if (c.Tok == token.GOTO && c.Label.Name == label && len(cc.Body) == 1) &&
				(i == len(sw.Body.List)-1 || !hadFallthrough) {
				idx = i
			}
			hadFallthrough = c.Tok == token.FALLTHROUGH
		default:
			if !canComplete(c) {
				hadFallthrough = false
				break
			}
			// Check that this is the final clause.
			if i != len(sw.Body.List)-1 {
				return -1
			}
			// If so, we need to add a fallthrough.
			if hasDefault && idx >= 0 {
				cc.Body = append(cc.Body, &ast.BranchStmt{Tok: token.FALLTHROUGH})
				return idx
			}
		}
	}
	if !hasDefault {
		return -1
	}
	return idx
}

// Inlines stmts into case clause i of the switch statement.
func inlineSwitchCase(sw *ast.SwitchStmt, i int, stmts []ast.Stmt) {
	cases := sw.Body.List
	// Replace the body with stmts.
	cc := cases[i].(*ast.CaseClause)
	cc.Body = append(cc.Body[:0], stmts...)
	// Move i to the end of the list, inplace.
	copy(cases[i:], cases[i+1:])
	cases[len(cases)-1] = cc
}

// Checks if s can possibly complete normally.
// It's OK to always return true.
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
	case *ast.SwitchStmt:
		var hasDefault bool
		for _, c := range s.Body.List {
			cc := c.(*ast.CaseClause)
			if len(cc.Body) == 0 || hasBreak(cc) || canComplete(cc.Body[len(cc.Body)-1]) {
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
