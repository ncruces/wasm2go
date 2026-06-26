package passes

import (
	"go/ast"
	"go/token"
)

// InlineSwitchGotos inlines switch cases that consist of a single goto,
// to an otherwise unused label.
func InlineSwitchGotos(fn *ast.FuncDecl) {
	uses := countBranches(fn)

	// This loop retries the optimization iteratively.
	for modified := true; modified; {
		modified = false
		postApplyStmts(fn, func(stmts []ast.Stmt) []ast.Stmt {
			for i := 0; i+1 < len(stmts); i++ {
				// A labeled statement, followed by an empty statement,
				// only used once as a (goto) target.
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
				id, fthrough := findSwitchCase(sw, ls.Label.Name)
				if id < 0 {
					continue
				}
				if !inlinableIntoSwitch(stmts[i+2:], uses) {
					continue
				}
				// Perform the optimization, and try again.
				inlineSwitchCase(sw, id, fthrough, stmts[i+2:])
				stmts = stmts[:i+1]
				modified = true
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
func findSwitchCase(sw *ast.SwitchStmt, label string) (idx int, fthrough bool) {
	// If such a case clause is found,
	// it must move to the end of the switch statement,
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
		if len(cc.Body) == 0 || canBreak(cc) {
			return -1, false
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
				return -1, false
			}
			// If so, we need to add a fallthrough.
			if hasDefault && idx >= 0 {
				fthrough = true
			}
		}
	}
	if !hasDefault {
		return -1, false
	}
	return idx, fthrough
}

// Checks if stmts can be safely inlined into a switch.
func inlinableIntoSwitch(stmts []ast.Stmt, branches map[string]int) bool {
	// Empty is inlinable.
	if len(stmts) == 0 {
		return true
	}

	// Fallthrough at the end would change meaning.
	// We could handle this, but it's not common enough to be worth it.
	if br, ok := stmts[len(stmts)-1].(*ast.BranchStmt); ok && br.Tok == token.FALLTHROUGH {
		return false
	}

	labels := set[string]{}
	for _, s := range stmts {
		// Collect top level labels.
		for ls, ok := s.(*ast.LabeledStmt); ok; {
			labels.add(ls.Label.Name)
			ls, ok = ls.Stmt.(*ast.LabeledStmt)
		}
		// Unlabeled break would change meaning.
		if canBreak(s) {
			return false
		}
	}

	// Can't have outside branches to those labels.
	if len(labels) > 0 {
		// Count local branches.
		local := countBranches(&ast.BlockStmt{List: stmts})
		for name := range labels {
			if branches[name] > local[name] {
				return false
			}
		}
	}
	return true
}

// Inlines stmts into case clause idx of the switch statement.
func inlineSwitchCase(sw *ast.SwitchStmt, idx int, fthrough bool, stmts []ast.Stmt) {
	cases := sw.Body.List
	// Replace the body with stmts.
	cc := cases[idx].(*ast.CaseClause)
	cc.Body = append(cc.Body[:0], stmts...)
	// Add fallthrough if needed.
	if fthrough {
		cc := cases[len(cases)-1].(*ast.CaseClause)
		cc.Body = append(cc.Body, &ast.BranchStmt{Tok: token.FALLTHROUGH})
	}
	// Move idx to the end of the list, inplace.
	copy(cases[idx:], cases[idx+1:])
	cases[len(cases)-1] = cc
}
