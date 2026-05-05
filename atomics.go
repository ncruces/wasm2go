package main

import (
	"fmt"
	"go/ast"
)

func (t *translator) readAtomicOpcode(fn *funcCompiler) error {
	code, err := readLEB128(t.in)
	if err != nil {
		return err
	}

	var offset uint64
	if code == 0x03 { // atomic.fence
		_, err = t.in.ReadByte() // 0x00
	} else {
		_, err = readLEB128(t.in) // align
		if err == nil {
			offset, err = readLEB128(t.in)
		}
	}
	if err != nil {
		return err
	}

	switch code {
	// case 0x00: // memory.atomic.notify
	// case 0x01: // memory.atomic.wait32
	// case 0x02: // memory.atomic.wait64
	// case 0x03: // atomic.fence

	case 0x10: // i32.atomic.load
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_load32")
		fn.push(convert(&ast.CallExpr{
			Fun:  newID("atomic_load32"),
			Args: []ast.Expr{&ast.SliceExpr{X: t.memory.selector, Low: addr}},
		}, "int32"))
	case 0x11: // i64.atomic.load
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_load64")
		fn.push(convert(&ast.CallExpr{
			Fun:  newID("atomic_load64"),
			Args: []ast.Expr{&ast.SliceExpr{X: t.memory.selector, Low: addr}},
		}, "int64"))
	case 0x12: // i32.atomic.load8_u
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_load8")
		fn.push(convert(&ast.CallExpr{
			Fun:  newID("atomic_load8"),
			Args: []ast.Expr{t.memory.selector, addr},
		}, "int32"))
	// case 0x13: // i32.atomic.load16_u
	case 0x14: // i64.atomic.load8_u
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_load8")
		fn.push(convert(&ast.CallExpr{
			Fun:  newID("atomic_load8"),
			Args: []ast.Expr{t.memory.selector, addr},
		}, "int64"))
	// case 0x15: // i64.atomic.load16_u
	case 0x16: // i64.atomic.load32_u
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_load32")
		fn.push(convert(&ast.CallExpr{
			Fun:  newID("atomic_load32"),
			Args: []ast.Expr{&ast.SliceExpr{X: t.memory.selector, Low: addr}},
		}, "int64"))

	case 0x17, 0x1d: // i32.atomic.store, i64.atomic.store32
		val := convert(fn.pop(), "uint32")
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_store32")
		fn.emit(&ast.ExprStmt{X: &ast.CallExpr{
			Fun:  newID("atomic_store32"),
			Args: []ast.Expr{&ast.SliceExpr{X: t.memory.selector, Low: addr}, val}}})
	case 0x18: // i64.atomic.store
		val := convert(fn.pop(), "uint64")
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_store64")
		fn.emit(&ast.ExprStmt{X: &ast.CallExpr{
			Fun:  newID("atomic_store64"),
			Args: []ast.Expr{&ast.SliceExpr{X: t.memory.selector, Low: addr}, val}}})
	case 0x19, 0x1b: // i32.atomic.store8, i64.atomic.store8
		val := convert(fn.pop(), "uint8")
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_store8")
		fn.emit(&ast.ExprStmt{X: &ast.CallExpr{
			Fun:  newID("atomic_store8"),
			Args: []ast.Expr{t.memory.selector, addr, val}}})
	// case 0x1a, 0x1c: // i32.atomic.store16, i64.atomic.store16

	case 0x1e: // i32.atomic.rmw.add
		val := convert(fn.pop(), "uint32")
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_add32")
		fn.push(convert(&ast.CallExpr{
			Fun: newID("atomic_add32"),
			Args: []ast.Expr{
				&ast.SliceExpr{X: t.memory.selector, Low: addr},
				val},
		}, "int32"))

	case 0x25: // i32.atomic.rmw.sub
		val := convert(fn.pop(), "uint32")
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_sub32")
		fn.push(convert(&ast.CallExpr{
			Fun: newID("atomic_sub32"),
			Args: []ast.Expr{
				&ast.SliceExpr{X: t.memory.selector, Low: addr},
				val},
		}, "int32"))

	case 0x41: // i32.atomic.rmw.xchg
		val := convert(fn.pop(), "uint32")
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_xchg32")
		fn.push(convert(&ast.CallExpr{
			Fun: newID("atomic_xchg32"),
			Args: []ast.Expr{
				&ast.SliceExpr{X: t.memory.selector, Low: addr},
				val},
		}, "int32"))

	case 0x43: // i32.atomic.rmw8.xchg_u
		val := convert(fn.pop(), "uint8")
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_xchg8")
		fn.push(convert(&ast.CallExpr{
			Fun:  newID("atomic_xchg8"),
			Args: []ast.Expr{t.memory.selector, addr, val},
		}, "int32"))

	case 0x45: // i64.atomic.rmw8.xchg_u
		val := convert(fn.pop(), "uint8")
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_xchg8")
		fn.push(convert(&ast.CallExpr{
			Fun:  newID("atomic_xchg8"),
			Args: []ast.Expr{t.memory.selector, addr, val},
		}, "int64"))

	case 0x48: // i32.atomic.rmw.cmpxchg
		new := convert(fn.pop(), "uint32")
		old := convert(fn.pop(), "uint32")
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_cmpxchg32")
		fn.push(convert(&ast.CallExpr{
			Fun: newID("atomic_cmpxchg32"),
			Args: []ast.Expr{
				&ast.SliceExpr{X: t.memory.selector, Low: addr},
				old, new},
		}, "int32"))

	case 0x4a: // i32.atomic.rmw8.cmpxchg_u
		new := convert(fn.pop(), "uint8")
		old := convert(fn.pop(), "uint8")
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_cmpxchg8")
		fn.push(convert(&ast.CallExpr{
			Fun:  newID("atomic_cmpxchg8"),
			Args: []ast.Expr{t.memory.selector, addr, old, new},
		}, "int32"))

	case 0x4c: // i64.atomic.rmw8.cmpxchg_u
		new := convert(fn.pop(), "uint8")
		old := convert(fn.pop(), "uint8")
		addr := fn.popAddr(offset)
		fn.helpers.add("atomic_cmpxchg8")
		fn.push(convert(&ast.CallExpr{
			Fun:  newID("atomic_cmpxchg8"),
			Args: []ast.Expr{t.memory.selector, addr, old, new},
		}, "int64"))

	default:
		return fmt.Errorf("unsupported opcode (atomic): 0xFE 0x%02X", code)
	}

	return nil
}
