package main

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/ncruces/wasm2go/util"
)

type funcCompiler struct {
	*translator

	typ  funcType
	call ast.Expr
	decl *ast.FuncDecl

	stack  stack[stackEntry]
	blocks stack[funcBlock]
	labels int
	temps  int
}

type entryKind int

const (
	entryConst entryKind = iota
	entryCond            // unevaluated condition
	entryExpr            // unevaluated expression
)

type stackEntry struct {
	kind entryKind
	expr ast.Expr
}

// Materializes all pending entryExpr and entryCond entries on the stack into temps.
func (fn *funcCompiler) flush() {
	for i := range fn.stack {
		e := &fn.stack[i]
		switch e.kind {
		case entryExpr:
			tmp := fn.newTempVal()
			fn.blocks.top().emit(&ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{tmp},
				Rhs: []ast.Expr{e.expr}})
			e.kind = entryConst
			e.expr = tmp
		case entryCond:
			tmp := fn.newTempVar()
			fn.blocks.top().emit(&ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{tmp},
							Type:  newID("int32")}}},
			}, &ast.IfStmt{
				Cond: e.expr,
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.AssignStmt{
							Tok: token.ASSIGN,
							Lhs: []ast.Expr{tmp},
							Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "1"}}}}}})
			e.kind = entryConst
			e.expr = tmp
		}
	}
}

// Emits statements to the current function.
func (fn *funcCompiler) emit(stmts ...ast.Stmt) {
	if len(stmts) > 0 {
		fn.flush()
		fn.blocks.top().emit(stmts...)
	}
}

// Returns a statement to exit n blocks.
// Materializes - but does not pop - the stack!
// After an unconditional branch, the code is unreachable,
// stack shape is irrelevant.
// After a conditional branch, values are supposed to stay on the stack
// for the next instruction.
func (fn *funcCompiler) branch(n uint64) (stmts []ast.Stmt) {
	if fn.blocks.top().unreachable {
		return nil
	}

	fn.flush()
	// Target block index.
	i := uint64(len(fn.blocks)) - n - 1

	// Returning from the function body.
	if i == 0 {
		ret := &ast.ReturnStmt{}
		for _, e := range fn.stack.last(len(fn.typ.results)) {
			ret.Results = append(ret.Results, e.expr)
		}
		return []ast.Stmt{ret}
	}

	blk := &fn.blocks[i]
	stmt := &ast.AssignStmt{Tok: token.ASSIGN}
	if blk.loopPos == 0 {
		// Breaking out of a block, set its results.
		stmt.Lhs = blk.results
	} else {
		// Breaking to the start of a loop, set its parameters.
		stmt.Lhs = blk.params
	}
	if len(stmt.Lhs) > 0 {
		for _, e := range fn.stack.last(len(stmt.Lhs)) {
			stmt.Rhs = append(stmt.Rhs, e.expr)
		}
		stmts = append(stmts, stmt)
	}

	// Create a label for the block we're jumping to.
	if blk.label == nil {
		blk.label = fn.newLabel()
	}

	return append(stmts, &ast.BranchStmt{Tok: token.GOTO, Label: blk.label})
}

// Returns an expression that loads a byte from memory (an l-value).
func (fn *funcCompiler) load8(offset uint64) ast.Expr {
	return &ast.IndexExpr{
		X:     fn.memory.selector,
		Index: fn.popAddr(offset)}
}

// Returns an expression that loads bytes from memory.
func (fn *funcCompiler) load(addr ast.Expr, typ string) (expr ast.Expr) {
	bits := typ[len(typ)-2:]

	// Load as unsigned, little-endian.
	fn.helpers.add("load" + bits)
	expr = &ast.CallExpr{
		Fun: newID("load" + bits),
		Args: []ast.Expr{&ast.SliceExpr{
			X:   fn.memory.selector,
			Low: addr}}}

	switch {
	case strings.HasPrefix(typ, "float"):
		// Convert to float.
		expr = &ast.CallExpr{
			Fun:  &ast.SelectorExpr{X: newID("math"), Sel: newID("Float" + bits + "frombits")},
			Args: []ast.Expr{expr}}
	case !strings.HasPrefix(typ, "u"):
		// Convert to signed, from unsigned.
		expr = convert(expr, typ)
	}
	return expr
}

// Returns a statement that stores bytes to memory.
func (fn *funcCompiler) store(addr, val ast.Expr, typ string) ast.Stmt {
	bits := typ[len(typ)-2:]

	if strings.HasPrefix(typ, "float") {
		// Convert to float.
		val = &ast.CallExpr{
			Fun:  &ast.SelectorExpr{X: newID("math"), Sel: newID("Float" + bits + "bits")},
			Args: []ast.Expr{val}}
	} else {
		// Convert to unsigned.
		val = convert(val, "uint"+bits)
	}

	// Store as unsigned, little-endian.
	fn.helpers.add("store" + bits)
	return &ast.ExprStmt{X: &ast.CallExpr{
		Fun: newID("store" + bits),
		Args: []ast.Expr{
			&ast.SliceExpr{
				X:   fn.memory.selector,
				Low: addr},
			val}}}
}

// Pushes expr (a literal, constant or materialized temporary) to the value stack.
func (fn *funcCompiler) pushConst(expr ast.Expr) {
	fn.stack.append(stackEntry{expr: expr, kind: entryConst})
}

// Pushes a pure expression (no observable side effects, including traps) to the value stack.
func (fn *funcCompiler) pushPure(expr ast.Expr) {
	if fn.blocks.top().unreachable {
		return
	}
	fn.stack.append(stackEntry{expr: expr, kind: entryExpr})
	if *noopt {
		fn.flush()
	}
}

// Pushes a pure condition (no observable side effects, including traps) to the value stack.
func (fn *funcCompiler) pushCond(cond ast.Expr) {
	if fn.blocks.top().unreachable {
		return
	}
	fn.stack.append(stackEntry{expr: cond, kind: entryCond})
	if *noopt {
		fn.flush()
	}
}

// Calls push or pushPure.
func (fn *funcCompiler) pushPureIf(pure bool, expr ast.Expr) {
	if pure {
		fn.pushPure(expr)
	} else {
		fn.push(expr)
	}
}

// Flushes the stack before pushing expr to the value stack.
func (fn *funcCompiler) push(expr ast.Expr) {
	if fn.blocks.top().unreachable {
		return
	}
	fn.flush()
	fn.pushPure(expr)
}

// Pops a value from the value stack.
func (fn *funcCompiler) pop() ast.Expr {
	if fn.blocks.top().unreachable {
		return &ast.BasicLit{Kind: token.INT, Value: "0"}
	}

	if fn.stack.top().kind == entryCond {
		fn.flush()
	}
	return fn.stack.pop().expr
}

// Pops a condition from the value stack.
func (fn *funcCompiler) popCond() ast.Expr {
	if fn.blocks.top().unreachable {
		return newID("false")
	}

	entry := fn.stack.pop()
	if entry.kind == entryCond {
		return entry.expr
	}

	return &ast.BinaryExpr{
		X: entry.expr, Op: token.NEQ,
		Y: &ast.BasicLit{Kind: token.INT, Value: "0"}}
}

// Pops an entry from the value stack.
func (fn *funcCompiler) popEntry() (entryKind, ast.Expr) {
	if fn.blocks.top().unreachable {
		return entryConst, &ast.BasicLit{Kind: token.INT, Value: "0"}
	}

	entry := fn.stack.pop()
	return entry.kind, entry.expr
}

// Pops an address from the stack, adds an offset, and returns it.
func (fn *funcCompiler) popAddr(offset uint64) (expr ast.Expr) {
	addr := fn.pop()

	if fn.memory.is64 {
		if offset == 0 {
			return addr
		}
		// Ensures wrap-around traps correctly.
		return &ast.BinaryExpr{
			Op: token.OR,
			X: &ast.BinaryExpr{
				Op: token.ADD,
				X:  addr,
				Y:  &ast.BasicLit{Kind: token.INT, Value: formatUint(offset)}},
			Y: &ast.BinaryExpr{
				Op: token.SHR,
				X:  addr,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "63"}}}
	}

	expr = convert(addr, "uint32")
	if offset == 0 {
		return expr
	}
	// Ensures wrap-around traps correctly.
	return &ast.BinaryExpr{
		Op: token.ADD,
		X:  convert(expr, "int64"),
		Y:  &ast.BasicLit{Kind: token.INT, Value: formatUint(offset)}}
}

// Executes a type conversion, first to types[0], then to types[1] and so on.
func (fn *funcCompiler) convert(types ...string) {
	fn.pushPure(convert(fn.pop(), types...))
}

// Executes a binary operator.
func (fn *funcCompiler) binOp(op token.Token) {
	fn.pushPureIf(op != token.QUO && op != token.REM,
		&ast.BinaryExpr{
			Y:  fn.pop(),
			X:  fn.pop(),
			Op: op})
}

// Executes a binary uint32 operator.
// Requires casts to unsigned and back.
func (fn *funcCompiler) binOpU32(op token.Token) {
	fn.pushPureIf(op != token.QUO && op != token.REM,
		convert(&ast.BinaryExpr{
			Y:  convert(fn.pop(), "uint32"),
			X:  convert(fn.pop(), "uint32"),
			Op: op,
		}, "int32"))
}

// Executes a binary uint64 operator.
// Requires casts to unsigned and back.
func (fn *funcCompiler) binOpU64(op token.Token) {
	fn.pushPureIf(op != token.QUO && op != token.REM,
		convert(&ast.BinaryExpr{
			Y:  convert(fn.pop(), "uint64"),
			X:  convert(fn.pop(), "uint64"),
			Op: op,
		}, "int64"))
}

// Executes a binary float64 operator.
// Requires casting the result,
// to avoid operations being combined against the Wasm spec.
func (fn *funcCompiler) binOpF64(op token.Token) {
	fn.pushPure(
		convert(&ast.BinaryExpr{
			Y:  fn.pop(),
			X:  fn.pop(),
			Op: op,
		}, "float64"))
}

// Executes a binary float32 operator.
// Requires casting the result,
// to avoid operations being combined against the Wasm spec.
func (fn *funcCompiler) binOpF32(op token.Token) {
	fn.pushPure(
		convert(&ast.BinaryExpr{
			Y:  fn.pop(),
			X:  fn.pop(),
			Op: op,
		}, "float32"))
}

// Executes a unary bitwise call.
func (fn *funcCompiler) bitOp(name string) {
	bits := name[len(name)-2:]

	fn.pushPure(
		convert(&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("bits"),
				Sel: newID(name)},
			Args: []ast.Expr{convert(fn.pop(), "uint"+bits)},
		}, "int"+bits))
}

// Executes a unary float64 math call.
func (fn *funcCompiler) uniMath64(name string) {
	fn.pushPure(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID(name)},
		Args: []ast.Expr{fn.pop()}})
}

// Executes a binary float64 math call.
func (fn *funcCompiler) binMath64(name string) {
	y := fn.pop()
	x := fn.pop()
	fn.pushPure(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID(name)},
		Args: []ast.Expr{x, y}})
}

// Executes a unary float32 math call.
func (fn *funcCompiler) uniMath32(name string) {
	fn.pushPure(
		convert(&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("math"),
				Sel: newID(name)},
			Args: []ast.Expr{convert(fn.pop(), "float64")},
		}, "float32"))
}

// Executes a Float32bits call.
func (fn *funcCompiler) float32bits() {
	fn.pushPure(
		convert(&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("math"),
				Sel: newID("Float32bits")},
			Args: []ast.Expr{fn.pop()},
		}, "int32"))
}

// Executes a Float64bits call.
func (fn *funcCompiler) float64bits() {
	fn.pushPure(
		convert(&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("math"),
				Sel: newID("Float64bits")},
			Args: []ast.Expr{fn.pop()},
		}, "int64"))
}

// Executes a Float32frombits call.
func (fn *funcCompiler) float32frombits() {
	fn.pushPure(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID("Float32frombits")},
		Args: []ast.Expr{convert(fn.pop(), "uint32")}})
}

// Executes a Float64frombits call.
func (fn *funcCompiler) float64frombits() {
	fn.pushPure(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID("Float64frombits")},
		Args: []ast.Expr{convert(fn.pop(), "uint64")}})
}

// Executes a unary helper call.
func (fn *funcCompiler) uniHelper(name string) {
	fn.helpers.add(name)
	fn.pushPureIf(pureHelpers.has(name),
		&ast.CallExpr{
			Fun:  newID(name),
			Args: []ast.Expr{fn.pop()}})
}

// Executes a binary helper call.
func (fn *funcCompiler) binHelper(name string) {
	fn.helpers.add(name)
	y := fn.pop()
	x := fn.pop()
	fn.pushPureIf(pureHelpers.has(name),
		&ast.CallExpr{
			Fun:  newID(name),
			Args: []ast.Expr{x, y}})
}

// Executes a signed division helper.
func (fn *funcCompiler) divHelper(typ string) {
	y := fn.pop()
	x := fn.pop()
	if v, ok := islit(y, typ); !ok {
		name := typ + "_div_s"
		fn.helpers.add(name)
		fn.push(&ast.CallExpr{Fun: newID(name), Args: []ast.Expr{x, y}})
	} else if v == -1 {
		name := typ + "_neg_s"
		fn.helpers.add(name)
		fn.push(&ast.CallExpr{Fun: newID(name), Args: []ast.Expr{x}})
	} else {
		fn.pushPureIf(v != 0, &ast.BinaryExpr{Op: token.QUO, X: x, Y: y})
	}
}

// Executes a bitwise shift helper.
func (fn *funcCompiler) bitHelper(name string) {
	typ, op, _ := strings.Cut(name, "_")
	y := fn.pop()
	x := fn.pop()

	v, ok := islit(y, typ)
	if !ok {
		fn.helpers.add(name)
		fn.pushPureIf(pureHelpers.has(name),
			&ast.CallExpr{
				Fun:  newID(name),
				Args: []ast.Expr{x, y}})
		return
	}

	if typ == "i32" {
		v &= 31
	}
	y = &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(v&63, 10)}
	var expr ast.Expr
	switch op {
	case "shl":
		expr = &ast.BinaryExpr{Op: token.SHL, X: x, Y: y}
	case "shr_s":
		expr = &ast.BinaryExpr{Op: token.SHR, X: x, Y: y}
	case "shr_u":
		s := "int32"
		u := "uint32"
		if typ == "i64" {
			s = "int64"
			u = "uint64"
		}
		expr = convert(&ast.BinaryExpr{Op: token.SHR, X: convert(x, u), Y: y}, s)
	}
	fn.pushPure(expr)
}

// Executes a binary builtin call.
func (fn *funcCompiler) binBuiltin(name string) {
	y := fn.pop()
	x := fn.pop()
	fn.pushPureIf(name == "min" || name == "max",
		&ast.CallExpr{
			Fun:  newID(name),
			Args: []ast.Expr{x, y}})
}

// Executes a zero equality comparison operator.
func (fn *funcCompiler) eqzOp() {
	kind, expr := fn.popEntry()
	// This is often used to negate conditions.
	if kind == entryCond {
		fn.pushCond(&ast.UnaryExpr{Op: token.NOT, X: expr})
	} else {
		fn.pushCond(&ast.BinaryExpr{
			X:  expr,
			Op: token.EQL,
			Y:  &ast.BasicLit{Kind: token.INT, Value: "0"}})
	}
}

// Executes a comparision operation.
func (fn *funcCompiler) cmpOp(op token.Token) {
	fn.pushCond(&ast.BinaryExpr{Y: fn.pop(), X: fn.pop(), Op: op})
}

// Executes a uint32 comparision operation.
// Requires casting to unsigned.
func (fn *funcCompiler) cmpOpU32(op token.Token) {
	fn.pushCond(&ast.BinaryExpr{
		Y:  convert(fn.pop(), "uint32"),
		X:  convert(fn.pop(), "uint32"),
		Op: op})
}

// Executes a uint64 comparision operation.
// Requires casting to unsigned.
func (fn *funcCompiler) cmpOpU64(op token.Token) {
	fn.pushCond(&ast.BinaryExpr{
		Y:  convert(fn.pop(), "uint64"),
		X:  convert(fn.pop(), "uint64"),
		Op: op})
}

func (fn *funcCompiler) newTempVal() *ast.Ident {
	id := tempVal(fn.temps)
	fn.temps++
	return id
}

func (fn *funcCompiler) newTempVar() *ast.Ident {
	id := tempVar(fn.temps)
	fn.temps++
	return id
}

func (fn *funcCompiler) newLabel() *ast.Ident {
	id := labelId(fn.labels)
	fn.labels++
	return id
}

func (fn *funcCompiler) cleanup() {
	ast.Inspect(fn.decl, fn.resolveImports)
	util.CheckMaterialized(fn.decl)
	util.RemoveUnusedLocals(fn.decl)
	if !*noopt {
		util.UnnestBlocks(fn.decl)
		util.RemoveEmptyStmts(fn.decl)
		util.RemoveSelfAssign(fn.decl)
		util.RemoveBlankAssigns(fn.decl)
		if util.RemoveReceiver(fn.decl) {
			fn.call.(*ast.ParenExpr).X = fn.decl.Name
		}
	}
}

type funcBlock struct {
	typ         funcType
	body        *ast.BlockStmt
	label       *ast.Ident
	ifStmt      *ast.IfStmt
	params      []ast.Expr
	results     []ast.Expr
	loopPos     int
	stackPos    int
	unreachable bool
	ifreachable bool
	elreachable bool
}

func (b *funcBlock) emit(stmts ...ast.Stmt) {
	if !b.unreachable {
		lst := &b.body.List
		*lst = append(*lst, stmts...)
	}
}

// Constructs a type conversion, first to types[0], then to types[1] and so on.
func convert(expr ast.Expr, types ...string) ast.Expr {
	for _, t := range types {
		expr = &ast.CallExpr{Fun: newID(t), Args: []ast.Expr{expr}}
	}
	return expr
}

func islit(expr ast.Expr, typ string) (int64, bool) {
	if call, ok := expr.(*ast.CallExpr); ok {
		if name, ok := call.Fun.(*ast.Ident); ok && name.Name == typ {
			if lit, ok := call.Args[0].(*ast.BasicLit); ok {
				if val, err := strconv.ParseInt(lit.Value, 0, 64); err == nil {
					return val, true
				}
			}
		}
	}
	return 0, false
}
