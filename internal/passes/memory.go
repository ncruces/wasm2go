package passes

import (
	"go/ast"
	"go/token"
	"slices"

	"golang.org/x/tools/go/ast/astutil"
)

// MemoryMode controls how a function loads the linear memory slice.
type MemoryMode int

const (
	// LoadOnRead loads the memory slice on every memory access.
	LoadOnRead MemoryMode = iota
	// LoadOnWrite stores the memory slice in a local variable and reloads it
	// after calls that may grow memory.
	LoadOnWrite
)

// SelectMemoryMode chooses between LoadOnRead and LoadOnWrite for fn.
//
// LoadOnWrite has an overhead of one load to initialize mem plus a reload
// after each call that may grow memory, so it is chosen when it reduces slice
// loads relative to LoadOnRead. A single read in a call-free function with a
// loop also uses LoadOnWrite so the load is hoisted out of the loop (when calls
// are present, a reload could itself run inside the loop).
func SelectMemoryMode(fn *ast.FuncDecl, slice ast.Expr, mayGrow func(*ast.CallExpr) bool) MemoryMode {
	reads, hasLoop, start, end := memorySpan(fn, slice)
	if reads == 0 {
		return LoadOnRead
	}

	calls := 0
	postApplyStmts(&ast.BlockStmt{List: fn.Body.List[start:end]}, func(stmts []ast.Stmt) []ast.Stmt {
		for _, s := range stmts {
			if stmtHasCall(s, mayGrow) {
				calls++
			}
		}
		return stmts
	})

	if calls == 0 && hasLoop {
		return LoadOnWrite
	}
	if 1+calls < reads {
		return LoadOnWrite
	}
	return LoadOnRead
}

// LoadMemoryOnWrite rewrites fn to store the memory slice in mem across its
// active span and reload it after calls that may grow memory.
//
// TODO: Elide unnecessary loads e.g. mem not read between calls.
func LoadMemoryOnWrite(fn *ast.FuncDecl, slice ast.Expr, mem *ast.Ident, mayGrow func(*ast.CallExpr) bool) {
	reads, _, start, end := memorySpan(fn, slice)
	if reads == 0 {
		return
	}

	region := &ast.BlockStmt{List: slices.Clone(fn.Body.List[start:end])}
	astutil.Apply(region, func(c *astutil.Cursor) bool {
		if u, ok := c.Node().(*ast.UnaryExpr); ok && u.Op == token.AND && u.X == slice {
			return false
		}
		if c.Node() == slice {
			c.Replace(mem)
			return false
		}
		return true
	}, nil)
	postApplyStmts(region, func(stmts []ast.Stmt) []ast.Stmt {
		for i := len(stmts) - 1; i >= 0; i-- {
			if stmtHasCall(stmts[i], mayGrow) {
				reload := &ast.AssignStmt{Tok: token.ASSIGN, Lhs: []ast.Expr{mem}, Rhs: []ast.Expr{slice}}
				stmts = slices.Insert(stmts, i+1, ast.Stmt(reload))
			}
		}
		return stmts
	})
	init := &ast.AssignStmt{Tok: token.DEFINE, Lhs: []ast.Expr{mem}, Rhs: []ast.Expr{slice}}
	fn.Body.List = slices.Concat(fn.Body.List[:start], []ast.Stmt{init}, region.List, fn.Body.List[end:])
}

// Bounds [start:end] to the top-level span of memory reads, labels, and gotos.
// Declaring mem at start skips leading guards and calls, prevents any goto from
// jumping over the declaration, and avoids reloading after calls in trailing
// read-free statements.
func memorySpan(fn *ast.FuncDecl, slice ast.Expr) (reads int, hasLoop bool, start, end int) {
	start = -1
	if fn.Body == nil {
		return 0, false, -1, 0
	}
	labels := set[string]{}
	for i, s := range fn.Body.List {
		r, br, loop := inspectMemStmt(s, slice, labels)
		reads += r
		hasLoop = hasLoop || loop
		if r > 0 || br {
			if start < 0 {
				start = i
			}
			end = i + 1
		}
	}
	return reads, hasLoop, start, end
}

// Inspects s for direct reads of the memory slice, labels or gotos, and backward-goto loops.
func inspectMemStmt(s ast.Stmt, slice ast.Expr, labels set[string]) (reads int, branch, loop bool) {
	ast.Inspect(s, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.UnaryExpr:
			return !(x.Op == token.AND && x.X == slice)
		case *ast.LabeledStmt:
			labels.add(x.Label.Name)
			branch = true
		case *ast.BranchStmt:
			if x.Tok == token.GOTO {
				branch = true
				loop = loop || labels.has(x.Label.Name)
			}
		}
		if n == slice {
			reads++
			return false
		}
		return true
	})
	return reads, branch, loop
}

// Reports whether s is an elementary statement containing a call matching pred.
// Compound statements are skipped because calls are materialized into their own
// ExprStmt or AssignStmt and visited at their inner statement list.
func stmtHasCall(s ast.Stmt, pred func(*ast.CallExpr) bool) (found bool) {
	for ls, ok := s.(*ast.LabeledStmt); ok; ls, ok = s.(*ast.LabeledStmt) {
		s = ls.Stmt
	}
	if !is[*ast.ExprStmt](s) && !is[*ast.AssignStmt](s) {
		return false
	}
	ast.Inspect(s, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok && pred(ce) {
			found = true
		}
		return !found
	})
	return found
}
