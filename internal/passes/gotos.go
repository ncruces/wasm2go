package passes

import (
	"go/ast"

	"golang.org/x/tools/go/ast/astutil"
)

func InlineSingleGotos(fn *ast.FuncDecl) {
	uses := countBranches(fn)

	for {
		var label string
		var block []ast.Stmt

		postApplyStmts(fn, func(stmts []ast.Stmt) []ast.Stmt {
			if len(label) > 0 {
				return stmts
			}
			for i := 0; i+1 < len(stmts)-1; i++ {
				// A labeled statement, followed by an empty statement,
				// only used once as a goto target.
				ls, ok := stmts[i+1].(*ast.LabeledStmt)
				if !ok || !is[*ast.EmptyStmt](ls.Stmt) || uses[ls.Label.Name] != 1 {
					continue
				}
				// Preceeded by and ending in a branch.
				if !endsInBranch(stmts[i]) || !startsWithBranch(stmts[len(stmts)-1]) {
					continue
				}
				// Validate the optimization.
				if !inlinableBlock(stmts[i+2:], uses) {
					continue
				}
				// Perform the optimization, and try again.
				label = ls.Label.Name
				block = stmts[i+2:]
				stmts = stmts[:i+1]
			}
			return stmts
		})
		if len(label) == 0 {
			break
		}

		var done bool
		astutil.Apply(fn, func(c *astutil.Cursor) bool {
			// Find the goto branch.
			bs, ok := c.Node().(*ast.BranchStmt)
			if !ok || bs.Label == nil || bs.Label.Name != label {
				return !done
			}
			// Replace it with the block.
			switch {
			case len(block) == 1:
				c.Replace(block[0])
			case c.Index() >= 0:
				for _, n := range block {
					c.InsertBefore(n)
				}
				c.Delete()
			default:
				c.Replace(&ast.BlockStmt{List: block})
			}
			done = true
			return false
		}, nil)
	}
}

func endsInBranch(s ast.Stmt) bool {
	switch s := s.(type) {
	case *ast.ReturnStmt, *ast.BranchStmt:
		return true
	case *ast.BlockStmt:
		if len := len(s.List); len > 0 {
			return endsInBranch(s.List[len-1])
		}
	}
	return false
}

func startsWithBranch(s ast.Stmt) bool {
	switch s := s.(type) {
	case *ast.ReturnStmt, *ast.BranchStmt:
		return true
	case *ast.BlockStmt:
		if len := len(s.List); len > 0 {
			return startsWithBranch(s.List[0])
		}
	}
	return false
}

func inlinableBlock(stmts []ast.Stmt, branches map[string]int) bool {
	if len(stmts) == 0 {
		return true
	}

	var unlabled bool
	for _, n := range stmts {
		ast.Inspect(n, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.ForStmt, *ast.RangeStmt, *ast.SelectStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
				return false
			case *ast.BranchStmt:
				if n.Label == nil {
					unlabled = true
					return false
				}
			}
			return !unlabled
		})
	}
	if unlabled {
		return false
	}

	labels := set[string]{}
	for _, s := range stmts {
		for ls, ok := s.(*ast.LabeledStmt); ok; {
			labels.add(ls.Label.Name)
			ls, ok = ls.Stmt.(*ast.LabeledStmt)
		}
	}

	if len(labels) > 0 {
		local := countBranches(&ast.BlockStmt{List: stmts})
		for name := range labels {
			if branches[name] > local[name] {
				return false
			}
		}
	}
	return true
}
