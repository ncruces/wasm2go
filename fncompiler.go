package main

import (
	"go/ast"
	"go/token"
	"strconv"
)

type funcCompiler struct {
	*translator

	typ  funcType
	decl *ast.FuncDecl

	top    ast.Expr
	cond   ast.Expr
	stack  []ast.Expr
	blocks []funcBlock
	labels int
	temps  int
}

// Returns a branch to block n instrution.
func (fn *funcCompiler) branch(n uint64) ast.Stmt {
	// Returning from the function body.
	if n == 0 {
		ret := &ast.ReturnStmt{}
		if len(fn.typ.results) > 0 {
			ret.Results = make([]ast.Expr, len(fn.typ.results))
			for i := len(ret.Results) - 1; i >= 0; i-- {
				ret.Results[i] = fn.pop()
			}
		}
		return ret
	}

	// Create a label for the block we're jumping to.
	targetBlk := &fn.blocks[n]
	if targetBlk.label == nil {
		targetBlk.label = fn.newLabel()
	}
	// If it's not a loop, set results.
	if targetBlk.loopPos == 0 {
		targetBlk.setResults(fn)
	}
	return &ast.BranchStmt{Tok: token.GOTO, Label: targetBlk.label}
}

// Returns a memory byte instruction: m.memory[start+offset]
func (fn *funcCompiler) memory8(offset uint64) ast.Expr {
	return &ast.IndexExpr{
		X: &ast.SelectorExpr{
			X:   newID("m"),
			Sel: newID("memory"),
		},
		Index: &ast.BinaryExpr{
			X: &ast.CallExpr{
				Fun:  newID("int"),
				Args: []ast.Expr{fn.pop()},
			},
			Op: token.ADD,
			Y: &ast.CallExpr{
				Fun: newID("int"),
				Args: []ast.Expr{&ast.BasicLit{
					Kind:  token.INT,
					Value: strconv.FormatUint(offset, 10),
				}},
			},
		},
	}
}

// Returns a memory index instruction: m.memory[start+offset:]
func (fn *funcCompiler) memoryN(offset uint64) ast.Expr {
	return &ast.SliceExpr{
		X: &ast.SelectorExpr{
			X:   newID("m"),
			Sel: newID("memory"),
		},
		Low: &ast.BinaryExpr{
			X: &ast.CallExpr{
				Fun:  newID("int"),
				Args: []ast.Expr{fn.pop()},
			},
			Op: token.ADD,
			Y: &ast.CallExpr{
				Fun: newID("int"),
				Args: []ast.Expr{&ast.BasicLit{
					Kind:  token.INT,
					Value: strconv.FormatUint(offset, 10),
				}},
			},
		},
	}
}

// Pushes expr (a literal, constant or materialized temporary) to the value stack.
func (fn *funcCompiler) pushConst(expr ast.Expr) {
	fn.stack = append(fn.stack, expr)
	fn.cond = nil
	fn.top = nil
}

// Pushes the materialization of expr to the value stack.
func (fn *funcCompiler) push(expr ast.Expr) {
	tmp := fn.newTempVar()
	blk := &fn.blocks[len(fn.blocks)-1]
	blk.append(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{tmp},
		Rhs: []ast.Expr{expr},
	})
	fn.pushConst(tmp)
	fn.top = expr
}

// Pushes the integer materialization of cond the value stack.
func (fn *funcCompiler) pushCond(cond ast.Expr) {
	tmp := fn.newTempVar()
	blk := &fn.blocks[len(fn.blocks)-1]
	blk.append(&ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{tmp},
					Type:  newID("int32"),
				},
			},
		},
	}, &ast.IfStmt{
		Cond: cond,
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: []ast.Expr{tmp},
					Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "1"}},
				},
			},
		},
	})
	fn.pushConst(tmp)
	fn.cond = cond
}

// Pops a materialized value from the value stack.
func (fn *funcCompiler) popCopy() ast.Expr {
	expr := pop(&fn.stack)
	fn.cond = nil
	fn.top = nil
	return expr
}

// Pops a (possibly unevaluated) value from the value stack.
// The value must be immediately used once and only once.
func (fn *funcCompiler) pop() ast.Expr {
	expr := pop(&fn.stack)
	top := fn.top
	fn.cond = nil
	fn.top = nil
	if top == nil {
		return expr
	}

	pop(&fn.blocks[len(fn.blocks)-1].body.List)
	return top
}

// Pops a (possibly unevaluated) condition from the value stack.
// The condition must be immediately used once and only once.
func (fn *funcCompiler) popCond() ast.Expr {
	expr := pop(&fn.stack)
	cond := fn.cond
	fn.cond = nil
	fn.top = nil
	if cond == nil {
		return &ast.BinaryExpr{
			X: expr, Op: token.NEQ,
			Y: &ast.BasicLit{Kind: token.INT, Value: "0"},
		}
	}

	pop(&fn.blocks[len(fn.blocks)-1].body.List)
	pop(&fn.blocks[len(fn.blocks)-1].body.List)
	return cond
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
		Op: op,
	})
}

// Executes a binary uint32 operator.
// Requires casts to unsigned and back.
func (fn *funcCompiler) binOpU32(op token.Token) {
	fn.push(&ast.CallExpr{
		Fun: newID("int32"),
		Args: []ast.Expr{&ast.BinaryExpr{
			Y: &ast.CallExpr{
				Fun:  newID("uint32"),
				Args: []ast.Expr{fn.pop()},
			},
			X: &ast.CallExpr{
				Fun:  newID("uint32"),
				Args: []ast.Expr{fn.pop()},
			},
			Op: op,
		}}})
}

// Executes a binary uint64 operator.
// Requires casts to unsigned and back.
func (fn *funcCompiler) binOpU64(op token.Token) {
	fn.push(&ast.CallExpr{
		Fun: newID("int64"),
		Args: []ast.Expr{&ast.BinaryExpr{
			Y: &ast.CallExpr{
				Fun:  newID("uint64"),
				Args: []ast.Expr{fn.pop()},
			},
			X: &ast.CallExpr{
				Fun:  newID("uint64"),
				Args: []ast.Expr{fn.pop()},
			},
			Op: op,
		}}})
}

// Executes a binary float64 operator.
// Requires casting the result,
// to avoid operations being combined against the Wasm spec.
func (fn *funcCompiler) binOpF64(op token.Token) {
	fn.push(&ast.CallExpr{
		Fun: newID("float64"),
		Args: []ast.Expr{&ast.BinaryExpr{
			Y:  fn.pop(),
			X:  fn.pop(),
			Op: op,
		}}})
}

// Executes a binary float32 operator.
// Requires casting the result,
// to avoid operations being combined against the Wasm spec.
func (fn *funcCompiler) binOpF32(op token.Token) {
	fn.push(&ast.CallExpr{
		Fun: newID("float32"),
		Args: []ast.Expr{&ast.BinaryExpr{
			Y:  fn.pop(),
			X:  fn.pop(),
			Op: op,
		}}})
}

// Executes a unary bitwise call.
func (fn *funcCompiler) bitOp(name string) {
	bits := name[len(name)-2:]

	fn.packages.add("math/bits")
	fn.push(&ast.CallExpr{
		Fun: newID("int" + bits),
		Args: []ast.Expr{&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("bits"),
				Sel: newID(name),
			},
			Args: []ast.Expr{&ast.CallExpr{
				Fun:  newID("uint" + bits),
				Args: []ast.Expr{fn.pop()},
			}},
		}},
	})
}

// Executes a unary float64 math call.
func (fn *funcCompiler) uniMath64(name string) {
	fn.packages.add("math")
	fn.push(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID(name),
		},
		Args: []ast.Expr{fn.pop()},
	})
}

// Executes a binary float64 math call.
func (fn *funcCompiler) binMath64(name string) {
	fn.packages.add("math")
	y := fn.pop()
	x := fn.pop()
	fn.push(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID(name),
		},
		Args: []ast.Expr{x, y},
	})
}

// Executes a unary float32 math call.
func (fn *funcCompiler) uniMath32(name string) {
	fn.packages.add("math")
	fn.push(&ast.CallExpr{
		Fun: newID("float32"),
		Args: []ast.Expr{&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("math"),
				Sel: newID(name),
			},
			Args: []ast.Expr{&ast.CallExpr{
				Fun:  newID("float64"),
				Args: []ast.Expr{fn.pop()},
			}},
		}},
	})
}

// Executes a binary float32 math call.
func (fn *funcCompiler) binMath32(name string) {
	fn.packages.add("math")
	y := fn.pop()
	x := fn.pop()
	fn.push(&ast.CallExpr{
		Fun: newID("float32"),
		Args: []ast.Expr{&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("math"),
				Sel: newID(name),
			},
			Args: []ast.Expr{
				&ast.CallExpr{Fun: newID("float64"), Args: []ast.Expr{x}},
				&ast.CallExpr{Fun: newID("float64"), Args: []ast.Expr{y}},
			},
		}},
	})
}

// Executes a Float32bits call.
func (fn *funcCompiler) float32bits() {
	fn.packages.add("math")
	fn.push(&ast.CallExpr{
		Fun: newID("int32"),
		Args: []ast.Expr{&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("math"),
				Sel: newID("Float32bits"),
			},
			Args: []ast.Expr{fn.pop()},
		}},
	})
}

// Executes a Float64bits call.
func (fn *funcCompiler) float64bits() {
	fn.packages.add("math")
	fn.push(&ast.CallExpr{
		Fun: newID("int64"),
		Args: []ast.Expr{&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   newID("math"),
				Sel: newID("Float64bits"),
			},
			Args: []ast.Expr{fn.pop()},
		}},
	})
}

// Executes a Float32frombits call.
func (fn *funcCompiler) float32frombits() {
	fn.packages.add("math")
	fn.push(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID("Float32frombits"),
		},
		Args: []ast.Expr{&ast.CallExpr{
			Fun:  newID("uint32"),
			Args: []ast.Expr{fn.pop()},
		}},
	})
}

// Executes a Float64frombits call.
func (fn *funcCompiler) float64frombits() {
	fn.packages.add("math")
	fn.push(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newID("math"),
			Sel: newID("Float64frombits"),
		},
		Args: []ast.Expr{&ast.CallExpr{
			Fun:  newID("uint64"),
			Args: []ast.Expr{fn.pop()},
		}},
	})
}

// Executes a unary helper call.
func (fn *funcCompiler) uniHelper(name string, pkgs ...string) {
	for _, p := range pkgs {
		fn.packages.add(p)
	}
	fn.helpers.add(name)

	fn.push(&ast.CallExpr{
		Fun:  newID(name),
		Args: []ast.Expr{fn.pop()},
	})
}

// Executes a binary helper call.
func (fn *funcCompiler) binHelper(name string, pkgs ...string) {
	for _, p := range pkgs {
		fn.packages.add(p)
	}
	fn.helpers.add(name)

	y := fn.pop()
	x := fn.pop()
	fn.push(&ast.CallExpr{
		Fun:  newID(name),
		Args: []ast.Expr{x, y},
	})
}

// Executes a binary builtin call.
func (fn *funcCompiler) binBuiltin(name string) {
	y := fn.pop()
	x := fn.pop()
	fn.push(&ast.CallExpr{
		Fun:  newID(name),
		Args: []ast.Expr{x, y},
	})
}

// Executes a zero equality comparison operator.
func (fn *funcCompiler) eqzOp() {
	fn.pushCond(&ast.BinaryExpr{
		X:  fn.pop(),
		Op: token.EQL,
		Y:  &ast.BasicLit{Kind: token.INT, Value: "0"},
	})
}

// Executes a comparision operation.
func (fn *funcCompiler) cmpOp(op token.Token) {
	fn.pushCond(&ast.BinaryExpr{Y: fn.pop(), X: fn.pop(), Op: op})
}

// Executes a uint32 comparision operation.
// Requires casting to unsigned.
func (fn *funcCompiler) cmpOpU32(op token.Token) {
	id := newID("uint32")
	fn.pushCond(&ast.BinaryExpr{
		Y: &ast.CallExpr{
			Fun:  id,
			Args: []ast.Expr{fn.pop()},
		},
		X: &ast.CallExpr{
			Fun:  id,
			Args: []ast.Expr{fn.pop()},
		},
		Op: op})
}

// Executes a uint64 comparision operation.
// Requires casting to unsigned.
func (fn *funcCompiler) cmpOpU64(op token.Token) {
	id := newID("uint64")
	fn.pushCond(&ast.BinaryExpr{
		Y: &ast.CallExpr{
			Fun:  id,
			Args: []ast.Expr{fn.pop()},
		},
		X: &ast.CallExpr{
			Fun:  id,
			Args: []ast.Expr{fn.pop()},
		},
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

type funcBlock struct {
	typ         funcType
	body        *ast.BlockStmt
	ifStmt      *ast.IfStmt
	results     []*ast.Ident
	label       *ast.Ident
	loopPos     int
	unreachable bool
}

func (b *funcBlock) append(stmts ...ast.Stmt) {
	lst := &b.body.List
	*lst = append(*lst, stmts...)
}

func (b *funcBlock) setResults(fn *funcCompiler) {
	if !b.unreachable {
		for i := len(b.results) - 1; i >= 0; i-- {
			b.append(&ast.AssignStmt{
				Lhs: []ast.Expr{b.results[i]},
				Rhs: []ast.Expr{fn.pop()},
				Tok: token.ASSIGN,
			})
		}
	}
}

// Constructs a type conversion, first to types[0], then to types[1] and so on.
func convert(x ast.Expr, types ...string) ast.Expr {
	for _, t := range types {
		x = &ast.CallExpr{Fun: newID(t), Args: []ast.Expr{x}}
	}
	return x
}

// Constructs a n bit memory load at an index.
func load(bits string, idx ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.SelectorExpr{
				X:   newID("binary"),
				Sel: newID("LittleEndian"),
			},
			Sel: newID("Uint" + bits),
		},
		Args: []ast.Expr{idx},
	}
}

// Constructs a n bit memory store at an index.
func store(bits string, idx ast.Expr, val ast.Expr) ast.Stmt {
	return &ast.ExprStmt{X: &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.SelectorExpr{
				X:   newID("binary"),
				Sel: newID("LittleEndian"),
			},
			Sel: newID("PutUint" + bits),
		},
		Args: []ast.Expr{idx, val},
	}}
}
