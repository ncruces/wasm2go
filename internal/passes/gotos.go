package passes

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/astutil"
)

func InlineSingleGotos(fn *ast.FuncDecl) {
	uses := countBranches(fn)

	for {
		var label string
		var block []ast.Stmt
		postApplyStmts(fn, func(stmts []ast.Stmt) ([]ast.Stmt, bool) {
			for i := 0; i+2 < len(stmts); i++ {
				// A labeled statement, followed by an empty statement,
				// only used once as a (goto) target.
				ls, ok := stmts[i+1].(*ast.LabeledStmt)
				if !ok || !is[*ast.EmptyStmt](ls.Stmt) || uses[ls.Label.Name] != 1 {
					continue
				}
				// Neither the previous statement,
				// nor the last to inline, can complete.
				if canComplete(stmts[i]) || canComplete(stmts[len(stmts)-1]) {
					continue
				}
				// Validate the optimization.
				if !inlinableBlock(stmts[i+2:], ls, uses) {
					continue
				}
				// Perform the optimization, and try again.
				label = ls.Label.Name
				block = stmts[i+2:]
				stmts = stmts[:i+1]
			}
			return stmts, label == ""
		})
		if label == "" {
			break
		}

		astutil.Apply(fn, nil, func(c *astutil.Cursor) bool {
			// Find the goto branch.
			bs, ok := c.Node().(*ast.BranchStmt)
			if !ok || bs.Label == nil || bs.Label.Name != label {
				return true
			}

			// Replace it with the block.
			var blk ast.BlockStmt
			blk.List = append(blk.List, block...)
			c.Replace(&blk)
			return false
		})
	}
}

func inlinableBlock(stmts []ast.Stmt, ls *ast.LabeledStmt, branches map[string]int) bool {
	// A fallthrough can't be moved.
	if bs, ok := stmts[len(stmts)-1].(*ast.BranchStmt); ok && bs.Tok == token.FALLTHROUGH {
		return false
	}

	// Count local branches.
	local := countBranches(&ast.BlockStmt{List: stmts})

	// Can't inline a block into itself.
	if _, ok := local[ls.Label.Name]; ok {
		return false
	}

	labels := set[string]{}
	for _, s := range stmts {
		// Collect top level labels.
		for ls, ok := s.(*ast.LabeledStmt); ok; {
			labels.add(ls.Label.Name)
			ls, ok = ls.Stmt.(*ast.LabeledStmt)
		}
		// Unlabeled branches might change meaning.
		if canBreak(s) || canContinue(s) {
			return false
		}
	}

	// Can't have outside branches to those labels.
	for name := range labels {
		if branches[name] > local[name] {
			return false
		}
	}
	return true
}
