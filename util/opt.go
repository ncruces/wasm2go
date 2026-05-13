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

// RemoveUnusedLocals replaces unused local variables with the blank identifier.
func RemoveUnusedLocals(n ast.Node) {
	uses := countUses(n)
	blank := ast.NewIdent("_")

	// If an identifer only shows up once,
	// in the left side of an definition,
	// replace it with the blank identifier.
	// If no names are being defined,
	// turn the definition into an assignment.
	astutil.Apply(n,
		func(c *astutil.Cursor) bool {
			if n, ok := c.Node().(*ast.AssignStmt); ok && n.Tok == token.DEFINE {
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
				return false
			}
			return true
		}, nil)
}

// RemoveSelfAssign removes self assignments to variables.
func RemoveSelfAssign(n ast.Node) {
	astutil.Apply(n, func(c *astutil.Cursor) bool {
		if n, ok := c.Node().(*ast.AssignStmt); ok && n.Tok == token.ASSIGN {
			var lhs, rhs []ast.Expr
			for i, expr := range n.Rhs {
				if idL, ok := n.Lhs[i].(*ast.Ident); ok {
					if idR, ok := expr.(*ast.Ident); ok && idL.Name == idR.Name {
						continue
					}
				}
				lhs = append(lhs, n.Lhs[i])
				rhs = append(rhs, expr)
			}

			if len(lhs) == 0 {
				c.Delete()
			} else if len(lhs) < len(n.Lhs) {
				n.Lhs = lhs
				n.Rhs = rhs
			}
			return true
		}
		return true
	}, nil)
}

// RemoveBlankAssigns removes redundant blank assignments from a variable
// when the variable is read elsewhere.
func RemoveBlankAssigns(n ast.Node) {
	uses := countUses(n)
	writes := countWrites(n)

	astutil.Apply(n, nil, func(c *astutil.Cursor) bool {
		if n, ok := c.Node().(*ast.AssignStmt); ok && n.Tok == token.ASSIGN {
			for _, expr := range n.Lhs {
				if id, ok := expr.(*ast.Ident); !ok || id.Name != "_" {
					return true
				}
			}

			var lhs, rhs []ast.Expr
			for i, expr := range n.Rhs {
				if id, ok := expr.(*ast.Ident); ok {
					if uses[id.Name]-writes[id.Name] > 1 {
						uses[id.Name]--
						continue
					}
				}
				lhs = append(lhs, n.Lhs[i])
				rhs = append(rhs, expr)
			}

			if len(lhs) == 0 {
				c.Delete()
			} else if len(lhs) < len(n.Lhs) {
				n.Lhs = lhs
				n.Rhs = rhs
			}
			return true
		}
		return true
	})
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

// UnnestBlocks removes block statements that do not contain variable declarations.
func UnnestBlocks(n ast.Node) {
	astutil.Apply(n, nil, func(c *astutil.Cursor) bool {
		// Only unnest if the node is part of a statement list.
		if c.Index() < 0 {
			return true
		}

		var blk *ast.BlockStmt
		var lbl *ast.LabeledStmt
		// Find blocks, and blocks preceded by labels.
		if b, ok := c.Node().(*ast.BlockStmt); ok {
			blk = b
		} else if l, ok := c.Node().(*ast.LabeledStmt); ok {
			if b, ok := l.Stmt.(*ast.BlockStmt); ok {
				blk = b
				lbl = l
			}
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
			if _, ok := s.(*ast.DeclStmt); ok {
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
			for i := len(blk.List) - 1; i >= 1; i-- {
				c.InsertAfter(blk.List[i])
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

// SimplifyGotos replaces all instances of "goto ${label}" with
// a return statement, specifically where labeled block matches:
//
//   - an empty labeled statement at the end of function body,
//     i.e. for valid AST this will be a function with no results
//
//   - a labeled statement consisting of only a "simple" return.
//     where fallthrough can be definitively ruled-out, the labeled
//     return will also be deleted from the funtion block.
func SimplifyGotos(fn *ast.FuncDecl) {
	type labeledReturn struct {
		Stmt   *ast.ReturnStmt
		NoFall bool
		Simple bool
		Uses   int
	}

	// Gather labeled return information by name.
	returns := make(map[string]*labeledReturn)
	mustGet := func(label string) *labeledReturn {
		ret, ok := returns[label]
		if !ok {
			ret = new(labeledReturn)
			returns[label] = ret
		}
		return ret
	}

	astutil.Apply(fn, nil, func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.LabeledStmt:
			switch s := n.Stmt.(type) {
			case *ast.ReturnStmt:
				// Gather information on
				// explicit labeled returns.
				ret := mustGet(n.Label.Name)
				ret.Simple = isSimpleReturn(s)
				prev := getPreviousInBlock(c.Parent(), n)
				ret.NoFall = cannotFallthrough(prev)
				ret.Stmt = s
			case *ast.EmptyStmt:
				// Also add a "deletion" marker for implicit
				// labeled returns, i.e. goto: function end.
				if n == fn.Body.List[len(fn.Body.List)-1] {
					ret := mustGet(n.Label.Name)
					ret.Simple = true
					ret.NoFall = true
					ret.Stmt = &ast.ReturnStmt{}
				}
			}
		case *ast.BranchStmt:
			if n.Tok == token.GOTO {
				// Count uses of all labels.
				ret := mustGet(n.Label.Name)
				ret.Uses++
			}
		}
		return true
	})

	// Filter out any without a return set,
	// i.e. labeled non-single-return blocks.
	for label, ret := range returns {
		if ret.Stmt == nil {
			delete(returns, label)
		}
	}

	astutil.Apply(fn, nil, func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.LabeledStmt:
			switch s := n.Stmt.(type) {
			case *ast.ReturnStmt:
				switch ret := mustGet(n.Label.Name); {
				// All returns that can't be fallen-into and are
				// either simple enough to not increase function
				// complexity if inlined, or are complex but only
				// used once, we remove and inline in the function.
				case ret.NoFall && (ret.Uses <= 1 || ret.Simple):
					c.Delete()

				// i.e. simple but can't definitively say that it
				// can't be fallen-through-to. we leave the return
				// in place but drop the label.
				case ret.Simple:
					c.Replace(s)
				}
			case *ast.EmptyStmt:
				// It's safe to remove all empty statement
				// labels we gathered, these will only be
				// implicit returns at end of function body.
				if _, ok := returns[n.Label.Name]; ok {
					c.Delete()
				}
			}
		case *ast.BranchStmt:
			if n.Tok == token.GOTO {
				// Replace all "goto ${label}" instances of those
				// labels replaced / deleted above, with just the
				// return statement originally contained at label.
				if ret, ok := returns[n.Label.Name]; //
				ok && (ret.Simple || ret.NoFall && ret.Uses <= 1) {
					c.Replace(ret.Stmt)
				}
			}
		}
		return true
	})
}

// getPreviousInBlock attempts to return the previous node to current 'n' in the given
// parent block 'p', i.e. searching for the previous node to 'n' in parent block's stmt list.
func getPreviousInBlock(p ast.Node, n ast.Stmt) ast.Node {
	switch p := p.(type) {
	case *ast.BlockStmt:
		i := slices.Index(p.List, n)
		if i > 0 {
			return p.List[i-1]
		} else {
			return nil
		}
	default:
		return nil
	}
}

// cannotFallthrough returns whether the given node 'n' will DEFINITIVELY
// NOT fallthrough to the stmt succeeding it. default return value is false.
func cannotFallthrough(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.LabeledStmt:
		return cannotFallthrough(n.Stmt)

	case *ast.ReturnStmt:
		return true

	case *ast.BranchStmt:
		switch n.Tok {
		// Anything other than GOTO / CONTINUE
		// alters the control flow in such a
		// way we can't guarantee it won't fall.
		case token.GOTO, token.CONTINUE:
			return true
		default:
			return false
		}

	case *ast.BlockStmt:
		// Empty block always
		// falls to successor.
		if len(n.List) == 0 {
			return false
		}

		// Check if last statement
		// in block falls through.
		last := n.List[len(n.List)-1]
		return cannotFallthrough(last)

	case *ast.SwitchStmt:
		var hasDefault bool

		// Check through all clauses.
		for _, n := range n.Body.List {
			cc, _ := n.(*ast.CaseClause)

			// Empty case always
			// falls to successor.
			if len(cc.List) == 0 {
				return false
			}

			// Check if any statements
			// in block can break-out.
			//
			// TODO?: this is quite limited
			// in its heuristic abilities here
			// without handling breaks separately.
			for _, n := range cc.List {
				if !cannotFallthrough(n) {
					return false
				}
			}

			if !hasDefault {
				// Check if case is default stmt.
				hasDefault = (len(cc.List) == 0)
			}
		}

		// None of cases fallthrough
		// only definitely doesn't if
		// there's a default case.
		return hasDefault

	default:
		return false
	}
}

// isSimpleReturn applies isSimpleValue() to each of the return's
// result expressions, returning true only if all are simple.
func isSimpleReturn(r *ast.ReturnStmt) bool {
	for _, res := range r.Results {
		if !isSimpleValue(res) {
			return false
		}
	}
	return true
}

// isSimpleValue returns whether given ast.Expr is
// a simple value type, i.e. just a variable or literal,
// taking int account any casting to primitive types.
func isSimpleValue(n ast.Expr) bool {
	switch n := n.(type) {
	case *ast.Ident:
		return true
	case *ast.BasicLit:
		return true
	case *ast.UnaryExpr:
		return isSimpleValue(n.X)
	case *ast.BinaryExpr:
		return isSimpleValue(n.X) &&
			isSimpleValue(n.Y)
	case *ast.CallExpr:
		if len(n.Args) != 1 {
			return false
		}
		id, ok := n.Fun.(*ast.Ident)
		if !ok {
			return false
		}
		switch id.Name {
		case "int", "int8", "int16", "int32", "int64", // signed integer types
			"uint", "uint8", "uint16", "uint32", "uint64", // unsigned integer types
			"float32", "float64", // float types
			"i32", "i64", "f32", "f64": // anti-const-folding helpers (see helpers/helpers.go)
			return isSimpleValue(n.Args[0])
		}
	}
	return false
}

// countUses counts uses of an identifier.
func countUses(n ast.Node) map[string]int {
	uses := make(map[string]int)
	ast.Inspect(n, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			uses[id.Name]++
			return false
		}
		return true
	})
	return uses
}

// countWrites counts writes to an identifier.
func countWrites(n ast.Node) map[string]int {
	writes := make(map[string]int)
	ast.Inspect(n, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.ValueSpec:
			for _, id := range x.Names {
				writes[id.Name]++
			}
		case *ast.AssignStmt:
			for i := range x.Lhs {
				if id, ok := x.Lhs[i].(*ast.Ident); ok {
					writes[id.Name]++
				}
			}
		}
		return true
	})
	return writes
}
