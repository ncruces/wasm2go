package passes

import (
	"go/ast"
	"go/token"
	"slices"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/cfg"
)

// RemoveDeadCode eliminates unreachable statements.
func RemoveDeadCode(fn *ast.FuncDecl) {
	g := cfg.New(fn.Body, callMayReturn)

	dead := set[ast.Node]{}
	for _, b := range g.Blocks {
		if !b.Live {
			for _, n := range b.Nodes {
				dead.add(n)
			}
		}
	}

	if len(dead) == 0 {
		return
	}

	astutil.Apply(fn.Body, nil, func(c *astutil.Cursor) bool {
		if dead.has(c.Node()) {
			if c.Index() < 0 {
				c.Replace(&ast.EmptyStmt{})
			} else {
				c.Delete()
			}
			// Don't traverse inside deleted nodes
			return true
		}
		return true
	})
}

// InlineSingleGotos replaces a goto with its target,
// if the label can only be reached by that goto,
// and the target is a simple terminating block.
func InlineSingleGotos(fn *ast.FuncDecl) {
	// The CFG package only creates a KindLabel blocks
	// for blocks that are the labeled targets
	// of branch statements.

	// If we find a labeled block with no successors
	// and a single reachable predecessor,
	// and a goto statement with a matching label,
	// we have a basic block that can't be reached
	// by any other means and which always terminates.

	g := cfg.New(fn.Body, callMayReturn)

	// Count the reachable predecessors of labeled blocks.
	preds := map[*cfg.Block]int{}
	for _, b := range g.Blocks {
		if b.Kind == cfg.KindUnreachable || !b.Live {
			continue
		}
		for _, s := range b.Succs {
			if s.Kind == cfg.KindLabel {
				preds[s]++
			}
		}
	}

	// Count the goto statements targeting each label.
	uses := countBranches(fn)

	// Identify labeled blocks to inline, nodes to delete.
	toInline := map[string][]ast.Node{}
	toDelete := set[ast.Node]{}
	for _, b := range g.Blocks {
		// Labeled block, no successors.
		if b.Kind != cfg.KindLabel || len(b.Succs) != 0 {
			continue
		}
		ls := b.Stmt.(*ast.LabeledStmt)

		// A single reachable predecessor, a single goto, only statements.
		if preds[b] != 1 || uses[ls.Label.Name] != 1 || slices.ContainsFunc(b.Nodes, func(n ast.Node) bool {
			_, ok := n.(ast.Stmt)
			return !ok
		}) {
			continue
		}

		toInline[ls.Label.Name] = b.Nodes
		toDelete.add(ls)
		for _, n := range b.Nodes {
			toDelete.add(n)
		}
	}

	// Nothing to do.
	if len(toInline) == 0 {
		return
	}

	// Delete nodes.
	astutil.Apply(fn.Body, nil, func(c *astutil.Cursor) bool {
		if n := c.Node(); toDelete.has(n) {
			if c.Index() < 0 {
				c.Replace(&ast.EmptyStmt{})
			} else {
				c.Delete()
			}
		}
		return true
	})

	// Reinsert them.
	astutil.Apply(fn.Body, nil, func(c *astutil.Cursor) bool {
		bs, ok := c.Node().(*ast.BranchStmt)
		if !ok || bs.Label == nil {
			return true
		}
		ls, ok := toInline[bs.Label.Name]
		if !ok {
			return true
		}
		if bs.Tok != token.GOTO {
			panic("want GOTO, got " + bs.Tok.String())
		}
		switch {
		case len(ls) == 1:
			c.Replace(ls[0])
		case c.Index() >= 0:
			for _, n := range ls {
				c.InsertBefore(n)
			}
			c.Delete()
		default:
			b := &ast.BlockStmt{}
			for _, n := range ls {
				b.List = append(b.List, n.(ast.Stmt))
			}
			c.Replace(b)
		}
		return true
	})
}
