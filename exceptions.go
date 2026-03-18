package main

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
)

func (fn *funcCompiler) try(bt funcType) {
	blk := fn.blocks.top()

	// Declare block results outside the block.
	results := make([]ast.Expr, len(bt.results))
	for i, t := range []byte(bt.results) {
		tmp := fn.newTempVar()
		results[i] = tmp
		fn.emit(&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{tmp},
						Type:  wasmType(t).ident()}}}})
	}
	// Ensure results are used.
	if len(results) > 0 {
		fn.emit(&ast.AssignStmt{
			Tok: token.ASSIGN,
			Lhs: slices.Repeat([]ast.Expr{newID("_")}, len(results)),
			Rhs: results})
	}

	childBlk := funcBlock{
		typ:         bt,
		results:     results,
		body:        &ast.BlockStmt{},
		stackPos:    len(fn.stack) - len(bt.params),
		unreachable: blk.unreachable,
		ifreachable: blk.unreachable,
		elreachable: blk.unreachable,
		iifeDepth:   blk.iifeDepth + 1,
		isTry:       true,
	}

	if len(bt.params) > 0 {
		lhs := make([]ast.Expr, len(bt.params))
		rhs := make([]ast.Expr, len(bt.params))
		for i := len(bt.params) - 1; i >= 0; i-- {
			lhs[i] = fn.newTempVar()
			rhs[i] = fn.pop()
		}
		childBlk.params = lhs
		for _, p := range lhs {
			fn.pushConst(p)
		}
		fn.emit(&ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: lhs,
			Rhs: rhs})
	}

	funcLit := &ast.FuncLit{
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{newID("jumpID")}, Type: newID("int")},
					{Names: []*ast.Ident{newID("caught")}, Type: &ast.StarExpr{X: newID("Exception")}},
				}}},
		Body: childBlk.body,
	}

	deferBody := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.AssignStmt{ // r := recover()
				Tok: token.DEFINE,
				Lhs: []ast.Expr{newID("r")},
				Rhs: []ast.Expr{&ast.CallExpr{Fun: newID("recover")}}},
			&ast.IfStmt{ // if r == nil { return }
				Cond: &ast.BinaryExpr{X: newID("r"), Op: token.EQL, Y: newID("nil")},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{}}}},
			// Hook for Chunk 4 where we match Exception types.
			&ast.ExprStmt{ // panic(r)
				X: &ast.CallExpr{Fun: newID("panic"), Args: []ast.Expr{newID("r")}}}}}
	childBlk.deferBody = deferBody

	childBlk.body.List = append(childBlk.body.List, &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.FuncLit{Type: &ast.FuncType{}, Body: deferBody}}})

	routingSwitch := &ast.SwitchStmt{Tag: newID("jump"), Body: &ast.BlockStmt{}}
	routingSwitch.Body.List = append(routingSwitch.Body.List, &ast.CaseClause{
		List: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "-1"}},
		Body: []ast.Stmt{&ast.ReturnStmt{}}}) // Bubble up outer function return

	for i := 1; i < len(fn.blocks); i++ {
		b := &fn.blocks[i]
		if b.iifeDepth == blk.iifeDepth { // Catch escapes to current scope
			if b.label == nil {
				b.label = fn.newLabel() // Assign a label proactively!
			}
			routingSwitch.Body.List = append(routingSwitch.Body.List, &ast.CaseClause{
				List: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(i)}},
				Body: []ast.Stmt{&ast.BranchStmt{Tok: token.GOTO, Label: b.label}}})
		}
	}

	var defaultBody []ast.Stmt
	if blk.iifeDepth == 0 {
		defaultBody = []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: newID("panic"), Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"unreachable"`}}}}}
	} else {
		defaultBody = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{newID("jump"), newID("nil")}}}
	}
	routingSwitch.Body.List = append(routingSwitch.Body.List, &ast.CaseClause{Body: defaultBody})

	childBlk.ifStmt = &ast.IfStmt{
		Init: &ast.AssignStmt{
			Tok: token.DEFINE, Lhs: []ast.Expr{newID("jump"), newID("caught")},
			Rhs: []ast.Expr{&ast.CallExpr{Fun: funcLit}}},
		Cond: &ast.BinaryExpr{X: newID("jump"), Op: token.NEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "0"}},
		Body: &ast.BlockStmt{List: []ast.Stmt{routingSwitch}}}
	fn.emit(childBlk.ifStmt)
	fn.blocks.append(childBlk)
}

func (fn *funcCompiler) catch(opcode byte) error {
	blk := fn.blocks.top()

	if blk.isTry {
		blk.setResults(fn)
		if !blk.unreachable {
			fn.emit(&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.BasicLit{Kind: token.INT, Value: "0"},
					newID("nil")}})
		}
		blk.isTry = false
		blk.iifeDepth-- // The catch body executes in the parent scope.

		blk.catchSwitch = &ast.SwitchStmt{
			Tag:  &ast.SelectorExpr{X: newID("caught"), Sel: newID("Tag")},
			Body: &ast.BlockStmt{},
		}
		blk.ifStmt.Else = &ast.IfStmt{
			Cond: &ast.BinaryExpr{X: newID("caught"), Op: token.NEQ, Y: newID("nil")},
			Body: &ast.BlockStmt{List: []ast.Stmt{blk.catchSwitch}},
		}

		blk.deferCatch = &ast.BlockStmt{}
		ifEx := &ast.IfStmt{
			Init: &ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{newID("ex"), newID("ok")},
				Rhs: []ast.Expr{&ast.TypeAssertExpr{
					X:    newID("r"),
					Type: &ast.StarExpr{X: newID("Exception")}}},
			},
			Cond: newID("ok"),
			Body: blk.deferCatch,
		}

		// Inject the dynamic matching block immediately prior to panic(r)
		lst := blk.deferBody.List
		blk.deferBody.List = append(lst[:len(lst)-1], ifEx, lst[len(lst)-1])

		blk.ifreachable = blk.unreachable
	} else {
		blk.setResults(fn)
		blk.ifreachable = blk.ifreachable && blk.unreachable
	}

	fn.stack = fn.stack[:blk.stackPos]
	blk.unreachable = blk.elreachable

	newBody := &ast.BlockStmt{}
	blk.body = newBody

	if opcode == 0x07 {
		tagIdx, err := readLEB128(fn.in)
		if err != nil {
			return err
		}
		tag := fn.tags[tagIdx]

		blk.deferCatch.List = append(blk.deferCatch.List, &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  &ast.SelectorExpr{X: newID("ex"), Sel: newID("Tag")},
				Op: token.EQL,
				Y: &ast.IndexExpr{
					X:     &ast.SelectorExpr{X: newID("m"), Sel: newID("tags")},
					Index: &ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(tagIdx, 10)}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.AssignStmt{Tok: token.ASSIGN, Lhs: []ast.Expr{newID("jumpID")}, Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "0"}}},
				&ast.AssignStmt{Tok: token.ASSIGN, Lhs: []ast.Expr{newID("caught")}, Rhs: []ast.Expr{newID("ex")}},
				&ast.ReturnStmt{},
			}},
		})

		blk.catchSwitch.Body.List = append(blk.catchSwitch.Body.List, &ast.CaseClause{
			List: []ast.Expr{&ast.IndexExpr{
				X:     &ast.SelectorExpr{X: newID("m"), Sel: newID("tags")},
				Index: &ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(tagIdx, 10)}}},
			Body: []ast.Stmt{newBody},
		})

		for i, p := range []byte(tag.typ.params) {
			tmp := fn.newTempVal()
			fn.emit(&ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{tmp},
				Rhs: []ast.Expr{&ast.TypeAssertExpr{
					X: &ast.IndexExpr{
						X:     &ast.SelectorExpr{X: newID("caught"), Sel: newID("Val")},
						Index: &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(i)}},
					Type: wasmType(p).ident()}}})
			fn.pushConst(tmp)
		}
	} else {
		blk.deferCatch.List = append(blk.deferCatch.List,
			&ast.AssignStmt{Tok: token.ASSIGN, Lhs: []ast.Expr{newID("jumpID")}, Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "0"}}},
			&ast.AssignStmt{Tok: token.ASSIGN, Lhs: []ast.Expr{newID("caught")}, Rhs: []ast.Expr{newID("ex")}},
			&ast.ReturnStmt{},
		)

		blk.catchSwitch.Body.List = append(blk.catchSwitch.Body.List, &ast.CaseClause{
			Body: []ast.Stmt{newBody},
		})
	}

	return nil
}

func (fn *funcCompiler) throw() error {
	tagIdx, err := readLEB128(fn.in)
	if err != nil {
		return err
	}
	fn.usesExn = true
	tag := fn.tags[tagIdx]

	vals := make([]ast.Expr, len(tag.typ.params))
	for i := len(vals) - 1; i >= 0; i-- {
		vals[i] = fn.pop()
	}

	fn.emit(&ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: newID("panic"),
			Args: []ast.Expr{
				&ast.UnaryExpr{
					Op: token.AND,
					X: &ast.CompositeLit{
						Type: newID("Exception"),
						Elts: []ast.Expr{
							&ast.KeyValueExpr{
								Key: newID("Tag"),
								Value: &ast.IndexExpr{
									X:     &ast.SelectorExpr{X: newID("m"), Sel: newID("tags")},
									Index: &ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(tagIdx, 10)}}},
							&ast.KeyValueExpr{
								Key: newID("Val"),
								Value: &ast.CompositeLit{
									Type: &ast.ArrayType{Elt: newID("any")},
									Elts: vals}}}}}}}})
	fn.blocks.top().unreachable = true

	return nil
}
