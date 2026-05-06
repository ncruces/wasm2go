package main

import (
	"fmt"
	"go/ast"
	"strings"
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
	case 0x00: // memory.atomic.notify
	case 0x01: // memory.atomic.wait32
	// case 0x02: // memory.atomic.wait64
	// case 0x03: // atomic.fence

	case 0x10: // i32.atomic.load
		fn.push(fn.atomicLoad("int32", "atomic_load32", offset))
	case 0x11: // i64.atomic.load
		fn.push(fn.atomicLoad("int64", "atomic_load64", offset))
	case 0x12: // i32.atomic.load8_u
		fn.push(fn.atomicLoad("int32", "atomic_load8", offset))
	case 0x13: // i32.atomic.load16_u
		fn.push(fn.atomicLoad("int32", "atomic_load16", offset))
	case 0x14: // i64.atomic.load8_u
		fn.push(fn.atomicLoad("int64", "atomic_load8", offset))
	case 0x15: // i64.atomic.load16_u
		fn.push(fn.atomicLoad("int64", "atomic_load16", offset))
	case 0x16: // i64.atomic.load32_u
		fn.push(fn.atomicLoad("int64", "atomic_load32", offset))

	case 0x17, 0x1d: // i32.atomic.store, i64.atomic.store32
		fn.emit(fn.atomicStore("atomic_store32", offset))
	case 0x18: // i64.atomic.store
		fn.emit(fn.atomicStore("atomic_store64", offset))
	case 0x19, 0x1b: // i32.atomic.store8, i64.atomic.store8
		fn.emit(fn.atomicStore("atomic_store8", offset))
	case 0x1a, 0x1c: // i32.atomic.store16, i64.atomic.store16
		fn.emit(fn.atomicStore("atomic_store16", offset))

	case 0x1e: // i32.atomic.rmw.add
		fn.push(fn.atomicRmw("int32", "atomic_add32", offset))
	case 0x1f: // i64.atomic.rmw.add
		fn.push(fn.atomicRmw("int64", "atomic_add64", offset))
	case 0x24: // i64.atomic.rmw32.add_u
		fn.push(fn.atomicRmw("int64", "atomic_add32", offset))

	case 0x25: // i32.atomic.rmw.sub
		fn.push(fn.atomicRmw("int32", "atomic_sub32", offset))
	case 0x26: // i64.atomic.rmw.sub
		fn.push(fn.atomicRmw("int64", "atomic_sub64", offset))
	case 0x2b: // i64.atomic.rmw32.sub_u
		fn.push(fn.atomicRmw("int64", "atomic_sub32", offset))

	case 0x41: // i32.atomic.rmw.xchg
		fn.push(fn.atomicRmw("int32", "atomic_xchg32", offset))
	case 0x42: // i64.atomic.rmw.xchg
		fn.push(fn.atomicRmw("int64", "atomic_xchg64", offset))
	case 0x43: // i32.atomic.rmw8.xchg_u
		fn.push(fn.atomicRmw("int32", "atomic_xchg8", offset))
	case 0x44: // i32.atomic.rmw16.xchg_u
		fn.push(fn.atomicRmw("int32", "atomic_xchg16", offset))
	case 0x45: // i64.atomic.rmw8.xchg_u
		fn.push(fn.atomicRmw("int64", "atomic_xchg8", offset))
	case 0x46: // i64.atomic.rmw16.xchg_u
		fn.push(fn.atomicRmw("int64", "atomic_xchg16", offset))
	case 0x47: // i64.atomic.rmw32.xchg_u
		fn.push(fn.atomicRmw("int64", "atomic_xchg32", offset))

	case 0x48: // i32.atomic.rmw.cmpxchg
		fn.push(fn.atomicCmpxchg("int32", "atomic_cmpxchg32", offset))
	case 0x49: // i64.atomic.rmw.cmpxchg
		fn.push(fn.atomicCmpxchg("int64", "atomic_cmpxchg64", offset))
	case 0x4a: // i32.atomic.rmw8.cmpxchg_u
		fn.push(fn.atomicCmpxchg("int32", "atomic_cmpxchg8", offset))
	case 0x4b: // i32.atomic.rmw16.cmpxchg_u
		fn.push(fn.atomicCmpxchg("int32", "atomic_cmpxchg16", offset))
	case 0x4c: // i64.atomic.rmw8.cmpxchg_u
		fn.push(fn.atomicCmpxchg("int64", "atomic_cmpxchg8", offset))
	case 0x4d: // i64.atomic.rmw16.cmpxchg_u
		fn.push(fn.atomicCmpxchg("int64", "atomic_cmpxchg16", offset))
	case 0x4e: // i64.atomic.rmw32.cmpxchg_u
		fn.push(fn.atomicCmpxchg("int64", "atomic_cmpxchg32", offset))

	default:
		return fmt.Errorf("unsupported opcode (atomic): 0xFE 0x%02X", code)
	}
	return nil
}

func (fn *funcCompiler) atomicLoad(typ, name string, offset uint64) ast.Expr {
	addr := fn.popAddr(offset)

	fn.helpers.add(name)
	return convert(&ast.CallExpr{
		Fun:  newID(name),
		Args: []ast.Expr{fn.memory.selector, addr},
	}, typ)
}

func (fn *funcCompiler) atomicStore(name string, offset uint64) ast.Stmt {
	val := fn.pop()
	addr := fn.popAddr(offset)
	idx := strings.IndexAny(name, "0123456789")
	val = convert(val, "uint"+name[idx:])

	fn.helpers.add(name)
	return &ast.ExprStmt{X: &ast.CallExpr{
		Fun:  newID(name),
		Args: []ast.Expr{fn.memory.selector, addr, val},
	}}
}

func (fn *funcCompiler) atomicRmw(typ, name string, offset uint64) ast.Expr {
	val := fn.pop()
	addr := fn.popAddr(offset)
	idx := strings.IndexAny(name, "0123456789")
	val = convert(val, "uint"+name[idx:])

	fn.helpers.add(name)
	return convert(&ast.CallExpr{
		Fun:  newID(name),
		Args: []ast.Expr{fn.memory.selector, addr, val},
	}, typ)
}

func (fn *funcCompiler) atomicCmpxchg(typ, name string, offset uint64) ast.Expr {
	new := fn.pop()
	old := fn.pop()
	addr := fn.popAddr(offset)
	idx := strings.IndexAny(name, "0123456789")
	new = convert(new, "uint"+name[idx:])
	old = convert(old, "uint"+name[idx:])

	fn.helpers.add(name)
	return convert(&ast.CallExpr{
		Fun:  newID(name),
		Args: []ast.Expr{fn.memory.selector, addr, old, new},
	}, typ)
}
