package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"slices"
)

func (t *translator) readCodeSection() error {
	numFuncs, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	importedFuncs := uint64(len(t.functions)) - numFuncs

	for i := range numFuncs {
		i += importedFuncs
		_, err := readLEB128(t.in)
		if err != nil {
			return err
		}

		err = t.readCodeForFunction(&t.functions[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *translator) readCodeForFunction(fn *funcCompiler) error {
	body := &ast.BlockStmt{}
	fn.translator = t
	fn.decl.Body = body
	fn.blocks = []funcBlock{{body: body}}

	numVars, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	// Declare local variables.
	// Parameters are predeclared locals.
	vars := make([]ast.Expr, 0, numVars)
	numLocals := len(fn.typ.params)
	for range numVars {
		n, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		typ, err := t.in.ReadByte()
		if err != nil {
			return err
		}

		ids := make([]*ast.Ident, n)
		for i := range int(n) {
			ids[i] = localVar(numLocals)
			vars = append(vars, ids[i])
			numLocals++
		}
		body.List = append(body.List, &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: ids,
						Type:  wasmType(typ).ident()}}}})
	}
	// Ensure local variables are used.
	if len(vars) > 0 {
		fn.emit(&ast.AssignStmt{
			Tok: token.ASSIGN,
			Lhs: slices.Repeat([]ast.Expr{newID("_")}, len(vars)),
			Rhs: vars})
	}

	for {
		opcode, err := t.in.ReadByte()
		if err != nil {
			return err
		}

		blk := fn.blocks.top()

		switch opcode {
		case 0x00: // unreachable
			if !blk.unreachable {
				fn.emit(&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun:  newID("panic"),
						Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"unreachable"`}}}})
				blk.unreachable = true
			}

		case 0x01: // nop

		case 0x02, 0x03, 0x04: // block, loop, if
			bt, err := t.readBlockType()
			if err != nil {
				return err
			}

			var cond ast.Expr
			if opcode == 0x04 { // if
				cond = fn.popCond()
			}

			childBlk := funcBlock{
				typ:         bt,
				body:        &ast.BlockStmt{},
				stackPos:    len(fn.stack) - len(bt.params),
				unreachable: blk.unreachable,
				ifreachable: blk.unreachable,
				elreachable: blk.unreachable,
			}

			// Declare block results outside the block.
			if n := len(bt.results); n > 0 {
				res := make([]ast.Expr, n)
				for i, t := range []byte(bt.results) {
					tmp := fn.newTempVar()
					res[i] = tmp
					fn.emit(&ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{tmp},
									Type:  wasmType(t).ident()}}}})
				}
				// Ensure results are used.
				fn.emit(&ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: slices.Repeat([]ast.Expr{newID("_")}, n),
					Rhs: res})
				childBlk.results = res
			}

			// Blocks can naturally consume their arguments.
			// Loops and ifs need to declare them outside the block so
			// they persist across iterations and are available to
			// both branches of the statement.
			if n := len(bt.params); n > 0 && opcode != 0x02 { // params, not a block
				lhs := make([]ast.Expr, n)
				rhs := make([]ast.Expr, n)
				for i := n - 1; i >= 0; i-- {
					if opcode == 0x03 { // loop
						lhs[i] = fn.newTempVar()
					} else {
						lhs[i] = fn.newTempVal()
					}
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

			var stmt ast.Stmt
			switch opcode {
			case 0x02: // block
				stmt = childBlk.body

			case 0x03: // loop
				// Remember the loop start position.
				// Bitwise not makes the zero value useful (not a loop).
				childBlk.loopPos = ^len(blk.body.List)
				stmt = childBlk.body

			case 0x04: // if
				// We need to remember the if statement
				// so we can attach an else branch.
				childBlk.ifStmt = &ast.IfStmt{Cond: cond, Body: childBlk.body}
				stmt = childBlk.ifStmt
			}

			fn.emit(stmt)
			fn.blocks.append(childBlk)

		case 0x05: // else
			// Set the results of the if branch.
			if n := len(blk.results); n > 0 {
				stmt := &ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: blk.results,
					Rhs: make([]ast.Expr, n),
				}
				for i := n - 1; i >= 0; i-- {
					stmt.Rhs[i] = fn.pop()
				}
				fn.emit(stmt)
			}
			// Reset polymorphic stacks.
			if blk.unreachable {
				fn.stack = fn.stack[:blk.stackPos]
			}
			// Push the if's arguments again, for the else branch.
			for _, p := range blk.params {
				fn.pushConst(p)
			}
			// Create a new block at the same level,
			// make it the else branch.
			blk.body = &ast.BlockStmt{}
			blk.ifStmt.Else = blk.body
			blk.ifreachable = blk.unreachable
			blk.unreachable = blk.elreachable

		case 0x0b: // end
			if len(fn.blocks) == 1 { // End of the function body.
				if n := len(fn.typ.results); n > 0 && !blk.unreachable {
					ret := &ast.ReturnStmt{Results: make([]ast.Expr, n)}
					for i := n - 1; i >= 0; i-- {
						ret.Results[i] = fn.pop()
					}
					fn.emit(ret)
				}
				fn.cleanup()
				return nil
			}

			// If this is an if statement with results and no else block,
			// we need an else statement that assigns the params to the results.
			if blk.ifStmt != nil && blk.ifStmt.Else == nil && len(blk.results) > 0 {
				blk.ifStmt.Else = &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.AssignStmt{
							Tok: token.ASSIGN,
							Lhs: blk.results,
							Rhs: blk.params}}}
			}

			// Set the block results.
			if n := len(blk.results); n > 0 {
				stmt := &ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: blk.results,
					Rhs: make([]ast.Expr, n),
				}
				for i := n - 1; i >= 0; i-- {
					stmt.Rhs[i] = fn.pop()
				}
				fn.emit(stmt)
			}
			// Reset polymorphic stacks.
			if blk.unreachable {
				fn.stack = fn.stack[:blk.stackPos]
			}
			// Push the results again, for the parent block.
			for _, r := range blk.results {
				fn.pushConst(r)
			}

			fn.blocks.pop()
			parent := fn.blocks.top()

			// Add the label if requested.
			if blk.label != nil {
				if blk.loopPos != 0 {
					// At the start for loops.
					parent.body.List[^blk.loopPos] = &ast.LabeledStmt{
						Stmt:  parent.body.List[^blk.loopPos],
						Label: blk.label}
				} else {
					// At the end for other block types.
					fn.emit(&ast.LabeledStmt{
						Stmt:  &ast.EmptyStmt{},
						Label: blk.label})
				}
			}

			// A parent block is unreachable at this point
			// if the end of this block is unreachable,
			// and the block is a loop (loops jump to the start),
			// or it has no end label to jump to,
			// and the block is not an if statement,
			// or both branches were unreachable.
			parent.unreachable = blk.unreachable &&
				(blk.loopPos != 0 || blk.label == nil &&
					(blk.ifStmt == nil || blk.ifreachable))

		case 0x0c: // br
			n, err := readLEB128(t.in)
			if err != nil {
				return err
			}

			fn.emit(fn.branch(n)...)
			blk.unreachable = true // After an uncoditional goto.

		case 0x0d: // br_if
			n, err := readLEB128(t.in)
			if err != nil {
				return err
			}

			// Conditional break.
			fn.emit(&ast.IfStmt{
				Cond: fn.popCond(),
				Body: &ast.BlockStmt{List: fn.branch(n)}})

		case 0x0e: // br_table
			numTargets, err := readLEB128(t.in)
			if err != nil {
				return err
			}

			sw := &ast.SwitchStmt{Tag: fn.pop(), Body: &ast.BlockStmt{}}

			// Group targets by their destination to consolidate case clauses.
			var targetMap = map[uint64]int{}
			for i := range numTargets {
				target, err := readLEB128(t.in)
				if err != nil {
					return err
				}

				// New target, add a new case clause.
				id, ok := targetMap[target]
				if !ok {
					id = len(sw.Body.List)
					targetMap[target] = id
					sw.Body.List = append(sw.Body.List,
						&ast.CaseClause{Body: fn.branch(target)})
				}

				caseExpr := &ast.BasicLit{Kind: token.INT, Value: formatUint(i)}
				caseClse := sw.Body.List[id].(*ast.CaseClause)
				caseClse.List = append(caseClse.List, caseExpr)
			}

			// Add the default target.
			target, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			id, ok := targetMap[target]
			if !ok {
				// New target, add a default clause.
				sw.Body.List = append(sw.Body.List,
					&ast.CaseClause{Body: fn.branch(target)})
			} else {
				// Make the existing clause the default,
				// move it to the end.
				caseClse := sw.Body.List[id].(*ast.CaseClause)
				copy(sw.Body.List[id:], sw.Body.List[id+1:])
				sw.Body.List[len(sw.Body.List)-1] = caseClse
				caseClse.List = nil
			}

			fn.emit(sw)
			blk.unreachable = true // After switch.

		case 0x10, 0x11, 0x12, 0x13: // call, call_indirect, return_call, return_call_indirect
			var fun ast.Expr
			var typ funcType

			switch opcode {
			case 0x10, 0x12: // call, return_call
				index, err := readLEB128(t.in)
				if err != nil {
					return err
				}

				target := &t.functions[index]
				fun = target.call
				typ = target.typ

			default: // call_indirect, return_call_indirect
				typeIdx, err := readLEB128(t.in)
				if err != nil {
					return err
				}
				tableIdx, err := readLEB128(t.in)
				if err != nil {
					return err
				}

				var tab ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.tables[tableIdx].id}
				if t.tables[tableIdx].imported {
					tab = &ast.StarExpr{X: tab}
				}

				idx := fn.pop()
				typ = t.types[typeIdx]
				fun = &ast.TypeAssertExpr{
					X:    &ast.IndexExpr{X: tab, Index: convert(idx, "uint")},
					Type: typ.toAST(false)}
			}

			args := make([]ast.Expr, len(typ.params))
			for i := len(args) - 1; i >= 0; i-- {
				args[i] = fn.pop()
			}

			call := &ast.CallExpr{Fun: fun, Args: args}

			switch opcode {
			case 0x12, 0x13: // return_call, return_call_indirect
				if len(typ.results) == 0 {
					fn.emit(&ast.ExprStmt{X: call}, &ast.ReturnStmt{})
				} else {
					fn.emit(&ast.ReturnStmt{Results: []ast.Expr{call}})
				}
				blk.unreachable = true // After an uncoditional return.
				warnReturnCall()

			default: // call, call_indirect
				switch n := len(typ.results); n {
				case 0:
					fn.emit(&ast.ExprStmt{X: call})
				case 1:
					fn.push(call)
				default:
					lhs := make([]ast.Expr, n)
					for i := range lhs {
						lhs[i] = fn.newTempVal()
					}
					fn.emit(&ast.AssignStmt{
						Lhs: lhs,
						Tok: token.DEFINE,
						Rhs: []ast.Expr{call}})
					for _, r := range lhs {
						fn.pushConst(r)
					}
				}
			}

		case 0x0f: // return
			if !blk.unreachable {
				n := len(fn.typ.results)
				ret := &ast.ReturnStmt{Results: make([]ast.Expr, n)}
				for i := n - 1; i >= 0; i-- {
					ret.Results[i] = fn.pop()
				}
				fn.emit(ret)
				blk.unreachable = true // After an uncoditional return.
			}

		case 0x1a: // drop
			if expr := fn.drop(); expr != nil {
				fn.emit(&ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: []ast.Expr{newID("_")},
					Rhs: []ast.Expr{expr},
				})
			}

		case 0x1b: // select
			cond := fn.popCond()
			tmp := fn.newTempVar()
			fn.emit(&ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{tmp},
				Rhs: []ast.Expr{fn.pop()},
			}, &ast.IfStmt{
				Cond: cond,
				Body: &ast.BlockStmt{
					List: []ast.Stmt{&ast.AssignStmt{
						Tok: token.ASSIGN,
						Lhs: []ast.Expr{tmp},
						Rhs: []ast.Expr{fn.pop()}}}},
			})
			fn.pushConst(tmp)

		case 0x1c: // select (typed)
			n, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			for range n {
				if _, err := t.in.ReadByte(); err != nil {
					return err
				}
			}

			cond := fn.popCond()
			if n == 0 {
				fn.emit(&ast.IfStmt{
					Cond: cond,
					Body: &ast.BlockStmt{},
				})
				break
			}

			vf := make([]ast.Expr, n)
			for i := int(n) - 1; i >= 0; i-- {
				vf[i] = fn.pop()
			}

			vt := make([]ast.Expr, n)
			for i := int(n) - 1; i >= 0; i-- {
				vt[i] = fn.pop()
			}

			tmp := make([]ast.Expr, n)
			for i := range tmp {
				tmp[i] = fn.newTempVar()
			}

			fn.emit(&ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: tmp,
				Rhs: vf,
			}, &ast.IfStmt{
				Cond: cond,
				Body: &ast.BlockStmt{
					List: []ast.Stmt{&ast.AssignStmt{
						Tok: token.ASSIGN,
						Lhs: tmp,
						Rhs: vt}}}})

			for _, t := range tmp {
				fn.pushConst(t)
			}

		case 0x20: // local.get
			i, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			fn.pushPure(localVar(i)) // Pure because assigning locals flushes.

		case 0x21: // local.set
			i, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			fn.emit(&ast.AssignStmt{
				Lhs: []ast.Expr{localVar(i)},
				Rhs: []ast.Expr{fn.pop()},
				Tok: token.ASSIGN})

		case 0x22: // local.tee
			i, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			fn.emit(&ast.AssignStmt{
				Lhs: []ast.Expr{localVar(i)},
				Rhs: []ast.Expr{fn.pop()},
				Tok: token.ASSIGN})
			fn.pushPure(localVar(i)) // Pure because assigning locals flushes.

		case 0x23: // global.get
			e, err := t.globalGet()
			if err != nil {
				return err
			}
			fn.push(e)

		case 0x24: // global.set
			i, err := readLEB128(t.in)
			if err != nil {
				return err
			}

			var lhs ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.globals[i].id}
			if t.globals[i].imported {
				lhs = &ast.StarExpr{X: lhs}
			}
			fn.emit(&ast.AssignStmt{
				Lhs: []ast.Expr{lhs},
				Rhs: []ast.Expr{fn.pop()},
				Tok: token.ASSIGN})

		case 0x25: // table.get
			i, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			var tab ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.tables[i].id}
			if t.tables[i].imported {
				tab = &ast.StarExpr{X: tab}
			}
			fn.push(&ast.IndexExpr{X: tab, Index: fn.pop()})

		case 0x26: // table.set
			i, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			var tab ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.tables[i].id}
			if t.tables[i].imported {
				tab = &ast.StarExpr{X: tab}
			}
			fn.emit(&ast.AssignStmt{
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{fn.pop()},
				Lhs: []ast.Expr{&ast.IndexExpr{X: tab, Index: fn.pop()}}})

		case 0x2c, 0x2d, 0x30, 0x31: // load8
			_, err := readLEB128(t.in) // align
			if err != nil {
				return err
			}
			offset, err := readLEB128(t.in)
			if err != nil { // offset
				return err
			}

			idx := fn.load8(offset)
			switch opcode {
			case 0x2c: // i32.load8_s
				fn.push(convert(idx, "int8", "int32"))
			case 0x2d: // i32.load8_u
				fn.push(convert(idx, "int32"))
			case 0x30: // i64.load8_s
				fn.push(convert(idx, "int8", "int64"))
			case 0x31: // i64.load8_u
				fn.push(convert(idx, "int64"))
			}

		case 0x28, 0x29, 0x2a, 0x2b, 0x2e, 0x2f, 0x32, 0x33, 0x34, 0x35: // load
			_, err := readLEB128(t.in) // align
			if err != nil {
				return err
			}
			offset, err := readLEB128(t.in)
			if err != nil { // offset
				return err
			}

			switch opcode {
			case 0x28: // i32.load
				fn.push(fn.load("int32", offset))
			case 0x29: // i64.load
				fn.push(fn.load("int64", offset))
			case 0x2a: // f32.load
				fn.push(fn.load("float32", offset))
			case 0x2b: // f64.load
				fn.push(fn.load("float64", offset))
			case 0x2e: // i32.load16_s
				fn.push(convert(fn.load("int16", offset), "int32"))
			case 0x2f: // i32.load16_u
				fn.push(convert(fn.load("uint16", offset), "int32"))
			case 0x32: // i64.load16_s
				fn.push(convert(fn.load("int16", offset), "int64"))
			case 0x33: // i64.load16_u
				fn.push(convert(fn.load("uint16", offset), "int64"))
			case 0x34: // i64.load32_s
				fn.push(convert(fn.load("int32", offset), "int64"))
			case 0x35: // i64.load32_u
				fn.push(convert(fn.load("uint32", offset), "int64"))
			}

		case 0x3a, 0x3c: // store8
			_, err := readLEB128(t.in) // align
			if err != nil {
				return err
			}
			offset, err := readLEB128(t.in)
			if err != nil { // offset
				return err
			}

			val := fn.pop()
			idx := fn.load8(offset) // an l-value
			fn.emit(&ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{idx},
				Rhs: []ast.Expr{convert(val, "byte")}})

		case 0x36, 0x37, 0x38, 0x39, 0x3b, 0x3d, 0x3e: // store
			_, err := readLEB128(t.in) // align
			if err != nil {
				return err
			}
			offset, err := readLEB128(t.in)
			if err != nil { // offset
				return err
			}

			switch opcode {
			case 0x36: // i32.store
				fn.emit(fn.store("int32", offset))
			case 0x37: // i64.store
				fn.emit(fn.store("int64", offset))
			case 0x38: // f32.store
				fn.emit(fn.store("float32", offset))
			case 0x39: // f64.store
				fn.emit(fn.store("float64", offset))
			case 0x3b: // i32.store16
				fn.emit(fn.store("int16", offset))
			case 0x3d: // i64.store16
				fn.emit(fn.store("int16", offset))
			case 0x3e: // i64.store32
				fn.emit(fn.store("int32", offset))
			}

		case 0x3f: // memory.size
			_, _ = readLEB128(t.in) // memory index
			fn.push(convert(
				&ast.BinaryExpr{
					X: &ast.CallExpr{
						Fun:  newID("len"),
						Args: []ast.Expr{t.memory.selector}},
					Op: token.SHR,
					Y:  literal16,
				}, t.memory.stype()))

		case 0x40: // memory.grow
			_, err := readLEB128(t.in) // memory index
			if err != nil {
				return err
			}
			if fn.memory.imported {
				fn.push(convert(&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.SelectorExpr{X: newID("m"), Sel: newID("memImp")},
						Sel: newID("Grow")},
					Args: []ast.Expr{
						convert(fn.pop(), "int64"),
						&ast.SelectorExpr{X: newID("m"), Sel: newID("maxMem")}}},
					t.memory.stype()))
			} else {
				name := "memory_grow"
				if t.memory.shared {
					name = "atomic_memory_grow"
					needsUnsafe("shared memory")
				}
				fn.helpers.add(name)
				fn.push(convert(&ast.CallExpr{
					Fun: newID(name),
					Args: []ast.Expr{
						&ast.UnaryExpr{Op: token.AND, X: t.memory.selector},
						convert(fn.pop(), "int64"),
						&ast.SelectorExpr{X: newID("m"), Sel: newID("maxMem")}}},
					t.memory.stype()))
			}

		case 0x41: // i32.const
			e, err := t.constI32()
			if err != nil {
				return err
			}
			fn.pushConst(e)

		case 0x42: // i64.const
			e, err := t.constI64()
			if err != nil {
				return err
			}
			fn.pushConst(e)

		case 0x43: // f32.const
			e, err := t.constF32()
			if err != nil {
				return err
			}
			fn.pushConst(e)

		case 0x44: // f64.const
			e, err := t.constF64()
			if err != nil {
				return err
			}
			fn.pushConst(e)

		case 0x45: // i32.eqz
			fn.eqzOp()
		case 0x46: // i32.eq
			fn.cmpOp(token.EQL)
		case 0x47: // i32.ne
			fn.cmpOp(token.NEQ)
		case 0x48: // i32.lt_s
			fn.cmpOp(token.LSS)
		case 0x49: // i32.lt_u
			fn.cmpOpU32(token.LSS)
		case 0x4a: // i32.gt_s
			fn.cmpOp(token.GTR)
		case 0x4b: // i32.gt_u
			fn.cmpOpU32(token.GTR)
		case 0x4c: // i32.le_s
			fn.cmpOp(token.LEQ)
		case 0x4d: // i32.le_u
			fn.cmpOpU32(token.LEQ)
		case 0x4e: // i32.ge_s
			fn.cmpOp(token.GEQ)
		case 0x4f: // i32.ge_u
			fn.cmpOpU32(token.GEQ)

		case 0x50: // i64.eqz
			fn.eqzOp()
		case 0x51: // i64.eq
			fn.cmpOp(token.EQL)
		case 0x52: // i64.ne
			fn.cmpOp(token.NEQ)
		case 0x53: // i64.lt_s
			fn.cmpOp(token.LSS)
		case 0x54: // i64.lt_u
			fn.cmpOpU64(token.LSS)
		case 0x55: // i64.gt_s
			fn.cmpOp(token.GTR)
		case 0x56: // i64.gt_u
			fn.cmpOpU64(token.GTR)
		case 0x57: // i64.le_s
			fn.cmpOp(token.LEQ)
		case 0x58: // i64.le_u
			fn.cmpOpU64(token.LEQ)
		case 0x59: // i64.ge_s
			fn.cmpOp(token.GEQ)
		case 0x5a: // i64.ge_u
			fn.cmpOpU64(token.GEQ)

		case 0x5b: // f32.eq
			fn.cmpOp(token.EQL)
		case 0x5c: // f32.ne
			fn.cmpOp(token.NEQ)
		case 0x5d: // f32.lt
			fn.cmpOp(token.LSS)
		case 0x5e: // f32.gt
			fn.cmpOp(token.GTR)
		case 0x5f: // f32.le
			fn.cmpOp(token.LEQ)
		case 0x60: // f32.ge
			fn.cmpOp(token.GEQ)

		case 0x61: // f64.eq
			fn.cmpOp(token.EQL)
		case 0x62: // f64.ne
			fn.cmpOp(token.NEQ)
		case 0x63: // f64.lt
			fn.cmpOp(token.LSS)
		case 0x64: // f64.gt
			fn.cmpOp(token.GTR)
		case 0x65: // f64.le
			fn.cmpOp(token.LEQ)
		case 0x66: // f64.ge
			fn.cmpOp(token.GEQ)

		case 0x67: // i32.clz
			fn.bitOp("LeadingZeros32")
		case 0x68: // i32.ctz
			fn.bitOp("TrailingZeros32")
		case 0x69: // i32.popcnt
			fn.bitOp("OnesCount32")
		case 0x6a: // i32.add
			fn.binOp(token.ADD)
		case 0x6b: // i32.sub
			fn.binOp(token.SUB)
		case 0x6c: // i32.mul
			fn.binOp(token.MUL)
		case 0x6d: // i32.div_s
			fn.divHelper("i32")
		case 0x6e: // i32.div_u
			fn.binOpU32(token.QUO)
		case 0x6f: // i32.rem_s
			fn.binOp(token.REM)
		case 0x70: // i32.rem_u
			fn.binOpU32(token.REM)
		case 0x71: // i32.and
			fn.binOp(token.AND)
		case 0x72: // i32.or
			fn.binOp(token.OR)
		case 0x73: // i32.xor
			fn.binOp(token.XOR)
		case 0x74: // i32.shl
			fn.bitHelper("i32_shl")
		case 0x75: // i32.shr_s
			fn.bitHelper("i32_shr_s")
		case 0x76: // i32.shr_u
			fn.bitHelper("i32_shr_u")
		case 0x77: // i32.rotl
			fn.binHelper("i32_rotl")
		case 0x78: // i32.rotr
			fn.binHelper("i32_rotr")

		case 0x79: // i64.clz
			fn.bitOp("LeadingZeros64")
		case 0x7a: // i64.ctz
			fn.bitOp("TrailingZeros64")
		case 0x7b: // i64.popcnt
			fn.bitOp("OnesCount64")
		case 0x7c: // i64.add
			fn.binOp(token.ADD)
		case 0x7d: // i64.sub
			fn.binOp(token.SUB)
		case 0x7e: // i64.mul
			fn.binOp(token.MUL)
		case 0x7f: // i64.div_s
			fn.divHelper("i64")
		case 0x80: // i64.div_u
			fn.binOpU64(token.QUO)
		case 0x81: // i64.rem_s
			fn.binOp(token.REM)
		case 0x82: // i64.rem_u
			fn.binOpU64(token.REM)
		case 0x83: // i64.and
			fn.binOp(token.AND)
		case 0x84: // i64.or
			fn.binOp(token.OR)
		case 0x85: // i64.xor
			fn.binOp(token.XOR)
		case 0x86: // i64.shl
			fn.bitHelper("i64_shl")
		case 0x87: // i64.shr_s
			fn.bitHelper("i64_shr_s")
		case 0x88: // i64.shr_u
			fn.bitHelper("i64_shr_u")
		case 0x89: // i64.rotl
			fn.binHelper("i64_rotl")
		case 0x8a: // i64.rotr
			fn.binHelper("i64_rotr")

		case 0x8b: // f32.abs
			fn.uniHelper("f32_abs")
		case 0x8c: // f32.neg
			fn.pushPure(&ast.UnaryExpr{Op: token.SUB, X: fn.pop()})
		case 0x8d: // f32.ceil
			fn.uniMath32("Ceil")
		case 0x8e: // f32.floor
			fn.uniMath32("Floor")
		case 0x8f: // f32.trunc
			fn.uniMath32("Trunc")
		case 0x90: // f32.nearest
			fn.uniMath32("RoundToEven")
		case 0x91: // f32.sqrt
			fn.uniMath32("Sqrt")
		case 0x92: // f32.add
			fn.binOpF32(token.ADD)
		case 0x93: // f32.sub
			fn.binOpF32(token.SUB)
		case 0x94: // f32.mul
			fn.binOpF32(token.MUL)
		case 0x95: // f32.div
			fn.binOpF32(token.QUO) // go.dev/issue/43577
		case 0x96: // f32.min
			if *nanbox {
				fn.binHelper("f32_min")
			} else {
				fn.binBuiltin("min")
			}
		case 0x97: // f32.max
			if *nanbox {
				fn.binHelper("f32_max")
			} else {
				fn.binBuiltin("max")
			}
		case 0x98: // f32.copysign
			fn.binHelper("f32_copysign")

		case 0x99: // f64.abs
			fn.uniMath64("Abs")
		case 0x9a: // f64.neg
			fn.pushPure(&ast.UnaryExpr{Op: token.SUB, X: fn.pop()})
		case 0x9b: // f64.ceil
			fn.uniMath64("Ceil")
		case 0x9c: // f64.floor
			fn.uniMath64("Floor")
		case 0x9d: // f64.trunc
			fn.uniMath64("Trunc")
		case 0x9e: // f64.nearest
			fn.uniMath64("RoundToEven")
		case 0x9f: // f64.sqrt
			fn.uniMath64("Sqrt")
		case 0xa0: // f64.add
			fn.binOpF64(token.ADD)
		case 0xa1: // f64.sub
			fn.binOpF64(token.SUB)
		case 0xa2: // f64.mul
			fn.binOpF64(token.MUL)
		case 0xa3: // f64.div
			fn.binOpF64(token.QUO) // go.dev/issue/43577
		case 0xa4: // f64.min
			if *nanbox {
				fn.binHelper("f64_min")
			} else {
				fn.binBuiltin("min")
			}
		case 0xa5: // f64.max
			if *nanbox {
				fn.binHelper("f64_max")
			} else {
				fn.binBuiltin("max")
			}
		case 0xa6: // f64.copysign
			fn.binMath64("Copysign")

		case 0xa7: // i32.wrap_i64
			fn.convert("int32")

		case 0xa8: // i32.trunc_f32_s
			fn.uniHelper("i32_trunc_f32_s")
		case 0xa9: // i32.trunc_f32_u
			fn.uniHelper("i32_trunc_f32_u")
		case 0xaa: // i32.trunc_f64_s
			fn.uniHelper("i32_trunc_f64_s")
		case 0xab: // i32.trunc_f64_u
			fn.uniHelper("i32_trunc_f64_u")

		case 0xac: // i64.extend_i32_s
			fn.convert("int64")
		case 0xad: // i64.extend_i32_u
			fn.convert("uint32", "int64")

		case 0xae: // i64.trunc_f32_s
			fn.uniHelper("i64_trunc_f32_s")
		case 0xaf: // i64.trunc_f32_u
			fn.uniHelper("i64_trunc_f32_u")
		case 0xb0: // i64.trunc_f64_s
			fn.uniHelper("i64_trunc_f64_s")
		case 0xb1: // i64.trunc_f64_u
			fn.uniHelper("i64_trunc_f64_u")

		case 0xb2: // f32.convert_i32_s
			fn.convert("float32")
		case 0xb3: // f32.convert_i32_u
			fn.convert("uint32", "float32")
		case 0xb4: // f32.convert_i64_s
			fn.convert("float32")
		case 0xb5: // f32.convert_i64_u
			fn.convert("uint64", "float32")
		case 0xb6: // f32.demote_f64
			fn.convert("float32")

		case 0xb7: // f64.convert_i32_s
			fn.convert("float64")
		case 0xb8: // f64.convert_i32_u
			fn.convert("uint32", "float64")
		case 0xb9: // f64.convert_i64_s
			fn.convert("float64")
		case 0xba: // f64.convert_i64_u
			fn.convert("uint64", "float64")
		case 0xbb: // f64.promote_f32
			fn.convert("float64")

		case 0xbc: // i32.reinterpret_f32
			fn.float32bits()
		case 0xbd: // i64.reinterpret_f64
			fn.float64bits()
		case 0xbe: // f32.reinterpret_i32
			fn.float32frombits()
		case 0xbf: // f64.reinterpret_i64
			fn.float64frombits()

		case 0xc0: // i32.extend8_s
			fn.convert("int8", "int32")
		case 0xc1: // i32.extend16_s
			fn.convert("int16", "int32")
		case 0xc2: // i64.extend8_s
			fn.convert("int8", "int64")
		case 0xc3: // i64.extend16_s
			fn.convert("int16", "int64")
		case 0xc4: // i64.extend32_s
			fn.convert("int32", "int64")

		case 0xd0: // ref.null
			_, err := t.in.ReadByte()
			if err != nil {
				return err
			}
			fn.pushConst(newID("nil"))
		case 0xd1: // ref.is_null
			fn.pushCond(&ast.BinaryExpr{
				X:  fn.pop(),
				Op: token.EQL,
				Y:  newID("nil")})
		case 0xd2: // ref.func
			index, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			fn.pushConst(t.functions[index].call)

		case 0xfb: // GC
			code, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			return fmt.Errorf("unsupported opcode (GC): 0xFB 0x%02X", code)

		case 0xfc: // FC extensions
			err := t.readOpcodeExtended(fn)
			if err != nil {
				return err
			}

		case 0xfd: // SIMD
			code, err := readLEB128(t.in)
			if err != nil {
				return err
			}
			return fmt.Errorf("unsupported opcode (SIMD): 0xFD 0x%02X", code)

		case 0xfe: // Atomics
			err := t.readOpcodeAtomic(fn)
			if err != nil {
				return err
			}

		default:
			return fmt.Errorf("unsupported opcode: 0x%02X", opcode)
		}
	}
}
