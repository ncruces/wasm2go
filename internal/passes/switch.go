package passes

import (
	"go/ast"
	"go/token"
)

// InlineSwitchTargets collapses br_table case blocks into its switch block.
// Wasm encodes 'branch targets' at the ends of nested blocks, so an N-way
// br_table arrives as N blocks, the innermost containing a switch of gotos:
//
//	{
//		{
//			switch x {
//			case 0:
//				goto l0
//			default:
//				goto l1
//			}
//		}
//	l0:
//		;
//		a()
//	}
//	l1:
//	;
//	b()
//
// becomes
//
//	switch x {
//	case 0:
//		a()
//		fallthrough
//	default:
//		b()
//	}
//
// Targets are inlined from the inside out, only when the dispatch's goto is
// the only goto to its label (Go forbids jumping into a case body). A label
// with other entries is left unchanged, and keeps every target outside it from
// inlining.
//
// After inlining, the now-empty enclosing blocks are removed by UnnestBlocks.
func InlineSwitchTargets(fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}
	gotos := map[string]int{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if br, ok := n.(*ast.BranchStmt); ok && br.Tok == token.GOTO {
			gotos[br.Label.Name]++
		}
		return true
	})

	var walk func(listPtr *[]ast.Stmt)
	walk = func(listPtr *[]ast.Stmt) {
		list := *listPtr
		for _, s := range list {
			switch s := s.(type) {
			case *ast.BlockStmt:
				walk(&s.List) // Children first
			case *ast.LabeledStmt:
				if b, ok := s.Stmt.(*ast.BlockStmt); ok {
					walk(&b.List)
				}
			case *ast.IfStmt:
				walk(&s.Body.List)
				if els, ok := s.Else.(*ast.BlockStmt); ok {
					walk(&els.List)
				}
			case *ast.SwitchStmt:
				for _, c := range s.Body.List {
					if cc, ok := c.(*ast.CaseClause); ok {
						walk(&cc.Body)
					}
				}
			}
		}
		// A target: a statement ending in the dispatch, an end label
		// with only the dispatch's goto, and the target's code after.
		for i := 0; i+1 < len(list); i++ {
			ls, ok := list[i+1].(*ast.LabeledStmt)
			if !ok || !isEmpty(ls.Stmt) || gotos[ls.Label.Name] != 1 {
				continue
			}
			sw := endSwitch(list[i])
			if sw == nil {
				continue
			}
			cc := gotoCase(sw, ls.Label.Name)
			if cc == nil {
				continue
			}
			// A fallthrough injected by an earlier inline targets the
			// clause right after it; moving that clause to the end would
			// silently retarget the fallthrough. Leave the target alone.
			if afterFallthrough(sw, cc) {
				continue
			}
			// NOTE: Copy the remainder of the segment as the truncated list's
			// spare capacity is reused by later passes' cursor edits.
			seg := append([]ast.Stmt(nil), list[i+2:]...)
			*listPtr = list[:i+1]
			inline(sw, cc, seg)
			// The segment may contain further dispatches of its own.
			walk(&cc.Body)
			return
		}
	}
	walk(&fn.Body.List)
}

// inline moves seg into cc and makes cc the last case for proper fallthrough.
func inline(sw *ast.SwitchStmt, cc *ast.CaseClause, seg []ast.Stmt) {
	cases := sw.Body.List
	for i, c := range cases {
		if c == ast.Stmt(cc) {
			cases = append(cases[:i], cases[i+1:]...)
			break
		}
	}
	if n := len(cases); n > 0 {
		if last := cases[n-1].(*ast.CaseClause); !terminates(last.Body) {
			last.Body = append(last.Body[:len(last.Body):len(last.Body)], &ast.BranchStmt{Tok: token.FALLTHROUGH})
		}
	}
	cc.Body = seg
	sw.Body.List = append(cases, cc)
}

// endSwitch returns the switch statement that ends s.
func endSwitch(s ast.Stmt) *ast.SwitchStmt {
	switch s := s.(type) {
	case *ast.SwitchStmt:
		if s.Init == nil {
			return s
		}
	case *ast.BlockStmt:
		if n := len(s.List); n > 0 {
			return endSwitch(s.List[n-1])
		}
	}
	return nil
}

// afterFallthrough reports whether the clause before cc ends in a
// fallthrough, making cc's position load-bearing.
func afterFallthrough(sw *ast.SwitchStmt, cc *ast.CaseClause) bool {
	for i, c := range sw.Body.List {
		if c != ast.Stmt(cc) {
			continue
		}
		if i == 0 {
			return false
		}
		prev, ok := sw.Body.List[i-1].(*ast.CaseClause)
		if !ok || len(prev.Body) == 0 {
			return false
		}
		br, ok := prev.Body[len(prev.Body)-1].(*ast.BranchStmt)
		return ok && br.Tok == token.FALLTHROUGH
	}
	return false
}

// gotoCase returns the case whose body is exactly `goto label`.
func gotoCase(sw *ast.SwitchStmt, label string) *ast.CaseClause {
	for _, c := range sw.Body.List {
		cc, ok := c.(*ast.CaseClause)
		if !ok || len(cc.Body) != 1 {
			continue
		}
		if br, ok := cc.Body[0].(*ast.BranchStmt); ok && br.Tok == token.GOTO && br.Label.Name == label {
			return cc
		}
	}
	return nil
}

func isEmpty(s ast.Stmt) bool {
	_, ok := s.(*ast.EmptyStmt)
	return ok
}

// terminates reports whether the list ends in a terminating statement,
// so the fallthrough can be omitted (vet would flag it as unreachable).
func terminates(body []ast.Stmt) bool {
	if len(body) == 0 {
		return false
	}
	return terminating(body[len(body)-1])
}

func terminating(s ast.Stmt) bool {
	switch s := s.(type) {
	case *ast.BranchStmt:
		return s.Tok == token.GOTO || s.Tok == token.FALLTHROUGH
	case *ast.ReturnStmt:
		return true
	case *ast.ExprStmt:
		call, ok := s.X.(*ast.CallExpr)
		if !ok {
			return false
		}
		id, ok := call.Fun.(*ast.Ident)
		return ok && id.Name == "panic"
	case *ast.BlockStmt:
		return terminates(s.List)
	case *ast.LabeledStmt:
		return terminating(s.Stmt)
	case *ast.IfStmt:
		return s.Else != nil && terminating(s.Body) && terminating(s.Else)
	case *ast.SwitchStmt:
		// A switch is considered terminating if 1) there is a default case and
		// 2) every case ends in a terminating statement or fallthrough.
		def := false
		for _, c := range s.Body.List {
			cc, ok := c.(*ast.CaseClause)
			if !ok {
				return false
			}
			if cc.List == nil {
				def = true
			}
			if n := len(cc.Body); n > 0 {
				if br, ok := cc.Body[n-1].(*ast.BranchStmt); ok && br.Tok == token.FALLTHROUGH {
					continue
				}
			}
			if !terminates(cc.Body) {
				return false
			}
		}
		return def
	}
	return false
}
