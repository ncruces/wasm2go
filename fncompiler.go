package main

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type funcCompiler struct {
	*translator

	typ  funcType
	decl *ast.FuncDecl

	cond   ast.Expr
	stack  stack[ast.Expr]
	blocks stack[funcBlock]
	labels int
	temps  int
}

// Emits statements to the current function.
func (fn *funcCompiler) emit(stmts ...ast.Stmt) {
	fn.blocks.top().emit(stmts...)
}

// Returns a statement to exit n blocks.
func (fn *funcCompiler) branch(n uint64) []ast.Stmt {
	// Target block index.
	i := uint64(len(fn.blocks)) - n - 1

	// Returning from the function body.
	if i == 0 {
		fn.cond = nil
		n := len(fn.typ.results)
		ret := &ast.ReturnStmt{}
		ret.Results = append(ret.Results, fn.stack.last(n)...)
		return []ast.Stmt{ret}
	}

	// Create a label for the block we're jumping to.
	blk := &fn.blocks[i]
	if blk.loopPos == 0 {
		// Breaking out of a block, set its results.
		blk.setResults(fn)
	} else if n := len(blk.params); n > 0 {
		// Breaking to the start of a loop, set its parameters.
		stmt := &ast.AssignStmt{Tok: token.ASSIGN, Lhs: blk.params}
		stmt.Rhs = append(stmt.Rhs, fn.stack.last(n)...)
		fn.stack = fn.stack[:blk.stackPos+n]
		fn.cond = nil
		fn.emit(stmt)
	}

	if fn.blocks.top().unreachable {
		return nil
	}

	if blk.label == nil {
		blk.label = fn.newLabel()
	}

	return []ast.Stmt{&ast.BranchStmt{Tok: token.GOTO, Label: blk.label}}
}

// Returns an expression that loads a byte from memory (an l-value).
func (fn *funcCompiler) load8(offset uint64) ast.Expr {
	return &ast.IndexExpr{
		X:     &ast.SelectorExpr{X: newID("m"), Sel: fn.memory.id},
		Index: fn.popAddr(offset)}
}

// Returns an expression that loads bytes from memory.
func (fn *funcCompiler) load(addr ast.Expr, typ string) (expr ast.Expr) {
	bits := typ[len(typ)-2:]

	// Load as unsigned, little-endian.
	expr = &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.SelectorExpr{
				X:   newID("binary"),
				Sel: newID("LittleEndian")},
			Sel: newID("Uint" + bits)},
		Args: []ast.Expr{&ast.SliceExpr{
			X:   &ast.SelectorExpr{X: newID("m"), Sel: fn.memory.id},
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
	return &ast.ExprStmt{X: &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.SelectorExpr{
				X:   newID("binary"),
				Sel: newID("LittleEndian")},
			Sel: newID("PutUint" + bits)},
		Args: []ast.Expr{
			&ast.SliceExpr{
				X:   &ast.SelectorExpr{X: newID("m"), Sel: fn.memory.id},
				Low: addr},
			val}}}
}

// Returns an expression that loads bytes from memory (an l-value).
func (fn *funcCompiler) loadUnsafe(addr ast.Expr, typ string) ast.Expr {
	var bytes string
	switch typ[len(typ)-2:] {
	case "16":
		bytes = "2"
	case "32":
		bytes = "4"
	case "64":
		bytes = "8"
	}

	return &ast.StarExpr{X: &ast.CallExpr{
		Fun: &ast.ParenExpr{X: &ast.StarExpr{X: newID(typ)}},
		Args: []ast.Expr{&ast.CallExpr{
			Fun: &ast.SelectorExpr{X: newID("unsafe"), Sel: newID("Pointer")},
			Args: []ast.Expr{&ast.CallExpr{
				Fun: &ast.StarExpr{X: &ast.ArrayType{
					Len: &ast.BasicLit{Kind: token.INT, Value: bytes},
					Elt: newID("byte")}},
				Args: []ast.Expr{&ast.SliceExpr{
					X:   &ast.SelectorExpr{X: newID("m"), Sel: fn.memory.id},
					Low: addr}}}}}}}}
}

// Returns a statement that stores bytes to memory.
func (fn *funcCompiler) storeUnsafe(addr, val ast.Expr, typ string) ast.Stmt {
	idx := fn.loadUnsafe(addr, typ) // an l-value

	if !strings.HasPrefix(typ, "float") {
		val = convert(val, typ)
	}

	return &ast.AssignStmt{
		Tok: token.ASSIGN,
		Lhs: []ast.Expr{idx},
		Rhs: []ast.Expr{val}}
}

// Pushes expr (a literal, constant or materialized temporary) to the value stack.
func (fn *funcCompiler) pushConst(expr ast.Expr) {
	fn.stack.append(expr)
	fn.cond = nil
}

// Pushes the materialization of expr to the value stack.
func (fn *funcCompiler) push(expr ast.Expr) {
	if fn.blocks.top().unreachable {
		fn.pushConst(&ast.BasicLit{Kind: token.INT, Value: "0"})
		return
	}

	tmp := fn.newTempVar()
	fn.emit(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{tmp},
		Rhs: []ast.Expr{expr}})
	fn.pushConst(tmp)
}

// Pushes the integer materialization of cond the value stack.
func (fn *funcCompiler) pushCond(cond ast.Expr) {
	tmp := fn.newTempVar()
	fn.emit(&ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{tmp},
					Type:  newID("int32")}}},
	}, &ast.IfStmt{
		Cond: cond,
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: []ast.Expr{tmp},
					Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "1"}}}}}})
	fn.pushConst(tmp)
	fn.cond = cond
}

// Pops a value from the value stack.
func (fn *funcCompiler) pop() ast.Expr {
	if blk := fn.blocks.top(); blk.unreachable {
		return &ast.BasicLit{Kind: token.INT, Value: "0"}
	}

	fn.cond = nil
	return fn.stack.pop()
}

// Pops a condition from the value stack.
// The condition must be immediately used once and only once.
func (fn *funcCompiler) popCond() ast.Expr {
	if blk := fn.blocks.top(); blk.unreachable {
		return newID("false")
	}

	expr := fn.stack.pop()
	cond := fn.cond
	fn.cond = nil
	if cond == nil {
		return &ast.BinaryExpr{
			X: expr, Op: token.NEQ,
			Y: &ast.BasicLit{Kind: token.INT, Value: "0"}}
	}

	lst := &fn.blocks.top().body.List
	*lst = (*lst)[:len(*lst)-2]
	return cond
}

// Pops an address from the stack, adds an offset, and returns it as a uint32.
func (fn *funcCompiler) popAddr(offset uint64) (expr ast.Expr) {
	expr = convert(fn.pop(), "uint32")
	if offset == 0 {
		return expr
	}
	return &ast.BinaryExpr{
		Op: token.ADD,
		X:  convert(expr, "int64"),
		Y:  &ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(offset, 10)}}
}

// Executes a type conversion, first to types[0], then to types[1] and so on.
func (fn *funcCompiler) convert(types ...string) {
	fn.push(convert(fn.pop(), types...))
}

// Executes a binary operator.
func (fn *funcCompiler) binOp(op token.Token) {
	fn.push(&ast.BinaryExpr{
		Y:  fn.pop(),
		X:  fn.pop(),
		Op: op})
}

// Executes a binary uint32 operator.
// Requires casts to unsigned and back.
func (fn *funcCompiler) binOpU32(op token.Token) {
	fn.push(convert(
		&ast.BinaryExpr{
			Y:  convert(fn.pop(), "uint32"),
			X:  convert(fn.pop(), "uint32"),
			Op: op,
		}, "int32"))
}

// Executes a binary uint64 operator.
// Requires casts to unsigned and back.
func (fn *funcCompiler) binOpU64(op token.Token) {
	fn.push(convert(
		&ast.BinaryExpr{
			Y:  convert(fn.pop(), "uint64"),
			X:  convert(fn.pop(), "uint64"),
			Op: op,
		}, "int64"))
}

// Executes a binary float64 operator.
// Requires casting the result,
// to avoid operations being combined against the Wasm spec.
func (fn *funcCompiler) binOpF64(op token.Token) {
	fn.push(convert(
		&ast.BinaryExpr{
			Y:  fn.pop(),
			X:  fn.pop(),
			Op: op,
		}, "float64"))
}

// Executes a binary float32 operator.
// Requires casting the result,
// to avoid operations being combined against the Wasm spec.
func (fn *funcCompiler) binOpF32(op token.Token) {
	fn.push(convert(
		&ast.BinaryExpr{
			Y:  fn.pop(),
			X:  fn.pop(),
			Op: op,
		}, "float32"))
}

// Executes a unary bitwise call.
func (fn *funcCompiler) bitOp(name string) {
	bits := name[len(name)-2:]

	fn.push(convert(
		&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("bits"),
				Sel: newID(name)},
			Args: []ast.Expr{convert(fn.pop(), "uint"+bits)},
		}, "int"+bits))
}

// Executes a unary float64 math call.
func (fn *funcCompiler) uniMath64(name string) {
	fn.push(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID(name)},
		Args: []ast.Expr{fn.pop()}})
}

// Executes a binary float64 math call.
func (fn *funcCompiler) binMath64(name string) {
	y := fn.pop()
	x := fn.pop()
	fn.push(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID(name)},
		Args: []ast.Expr{x, y}})
}

// Executes a unary float32 math call.
func (fn *funcCompiler) uniMath32(name string) {
	fn.push(convert(
		&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("math"),
				Sel: newID(name)},
			Args: []ast.Expr{convert(fn.pop(), "float64")},
		}, "float32"))
}

// Executes a Float32bits call.
func (fn *funcCompiler) float32bits() {
	fn.push(convert(
		&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("math"),
				Sel: newID("Float32bits")},
			Args: []ast.Expr{fn.pop()},
		}, "int32"))
}

// Executes a Float64bits call.
func (fn *funcCompiler) float64bits() {
	fn.push(convert(
		&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("math"),
				Sel: newID("Float64bits")},
			Args: []ast.Expr{fn.pop()},
		}, "int64"))
}

// Executes a Float32frombits call.
func (fn *funcCompiler) float32frombits() {
	fn.push(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID("Float32frombits")},
		Args: []ast.Expr{convert(fn.pop(), "uint32")}})
}

// Executes a Float64frombits call.
func (fn *funcCompiler) float64frombits() {
	fn.push(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID("Float64frombits")},
		Args: []ast.Expr{convert(fn.pop(), "uint64")}})
}

// Executes a unary helper call.
func (fn *funcCompiler) uniHelper(name string) {
	fn.helpers.add(name)
	fn.push(&ast.CallExpr{
		Fun:  newID(name),
		Args: []ast.Expr{fn.pop()}})
}

// Executes a binary helper call.
func (fn *funcCompiler) binHelper(name string) {
	fn.helpers.add(name)
	y := fn.pop()
	x := fn.pop()
	fn.push(&ast.CallExpr{
		Fun:  newID(name),
		Args: []ast.Expr{x, y}})
}

// Executes a binary builtin call.
func (fn *funcCompiler) binBuiltin(name string) {
	y := fn.pop()
	x := fn.pop()
	fn.push(&ast.CallExpr{
		Fun:  newID(name),
		Args: []ast.Expr{x, y}})
}

// Executes a zero equality comparison operator.
func (fn *funcCompiler) eqzOp() {
	fn.pushCond(&ast.BinaryExpr{
		X:  fn.pop(),
		Op: token.EQL,
		Y:  &ast.BasicLit{Kind: token.INT, Value: "0"}})
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
	body := fn.blocks[0].body

	// Add Go imports.
	ast.Inspect(body, fn.resolveImports)

	// Count identifier uses.
	uses := make(map[string]int)
	ast.Inspect(body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			uses[id.Name]++
		}
		return true
	})

	astutil.Apply(body,
		// If an identifer only shows up in the left side of an assignment,
		// replace it with the blank identifier.
		func(c *astutil.Cursor) bool {
			if n, ok := c.Node().(*ast.AssignStmt); ok && n.Tok == token.DEFINE {
				anyUsed := false
				for i := range n.Lhs {
					if id, ok := n.Lhs[i].(*ast.Ident); ok {
						if uses[id.Name] == 1 {
							n.Lhs[i] = newID("_")
						} else if id.Name != "_" {
							anyUsed = true
						}
					}
				}
				if !anyUsed {
					n.Tok = token.ASSIGN
				}
			}
			return true
		},
		// Remove labels followed by an empty statement (;).
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

func (b *funcBlock) setResults(fn *funcCompiler) {
	if b.unreachable || len(b.results) == 0 {
		return
	}

	// Don't pop results from the stack.
	// If the branch is conditional,
	// the values are supposed to stay on the stack
	// for the next instruction.

	stmt := &ast.AssignStmt{
		Tok: token.ASSIGN,
		Lhs: b.results,
	}
	stmt.Rhs = append(stmt.Rhs, fn.stack.last(len(b.results))...)
	fn.cond = nil
	fn.emit(stmt)
}

// Constructs a type conversion, first to types[0], then to types[1] and so on.
func convert(expr ast.Expr, types ...string) ast.Expr {
	for _, t := range types {
		expr = &ast.CallExpr{Fun: newID(t), Args: []ast.Expr{expr}}
	}
	return expr
}

func iszero(expr ast.Expr) bool {
	if call, ok := expr.(*ast.CallExpr); ok {
		if name, ok := call.Fun.(*ast.Ident); ok && name.Name == "i32_const" {
			if len(call.Args) == 1 {
				lit, ok := call.Args[0].(*ast.BasicLit)
				return ok && lit.Kind == token.INT && lit.Value == "0"
			}
		}
	}
	return false
}
