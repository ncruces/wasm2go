package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

func (t *translator) readOpcodeExtended(fn *funcCompiler) error {
	code, err := readLEB128(t.in)
	if err != nil {
		return err
	}
	switch code {
	case 0x00: // i32.trunc_sat_f32_s
		fn.uniHelper("i32_trunc_sat_f32_s")
	case 0x01: // i32.trunc_sat_f32_u
		fn.uniHelper("i32_trunc_sat_f32_u")
	case 0x02: // i32.trunc_sat_f64_s
		fn.uniHelper("i32_trunc_sat_f64_s")
	case 0x03: // i32.trunc_sat_f64_u
		fn.uniHelper("i32_trunc_sat_f64_u")
	case 0x04: // i64.trunc_sat_f32_s
		fn.uniHelper("i64_trunc_sat_f32_s")
	case 0x05: // i64.trunc_sat_f32_u
		fn.uniHelper("i64_trunc_sat_f32_u")
	case 0x06: // i64.trunc_sat_f64_s
		fn.uniHelper("i64_trunc_sat_f64_s")
	case 0x07: // i64.trunc_sat_f64_u
		fn.uniHelper("i64_trunc_sat_f64_u")

	case 0x08: // memory.init
		i, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		_, err = readLEB128(t.in)
		if err != nil {
			return err
		}
		n := convert(fn.pop(), "uint32")
		src := convert(fn.pop(), "uint32")
		dst := convert(fn.pop(), t.memory.utype())
		fn.helpers.add("memory_init")
		fn.emit(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: newID("memory_init"),
				Args: []ast.Expr{
					t.memory.selector,
					t.dataExpr(int(i)), dst, src, n}}})

	case 0x09: // data.drop
		// No-op since data segments are constants.
		_, err := readLEB128(t.in)
		if err != nil {
			return err
		}

	case 0x0a: // memory.copy
		_, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		_, err = readLEB128(t.in)
		if err != nil {
			return err
		}
		typ := t.memory.utype()
		n := convert(fn.pop(), typ)
		src := convert(fn.pop(), typ)
		dst := convert(fn.pop(), typ)
		fn.helpers.add("memory_copy")
		fn.emit(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: newID("memory_copy"),
				Args: []ast.Expr{
					t.memory.selector,
					dst, src, n}}})

	case 0x0b: // memory.fill
		_, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		typ := t.memory.utype()
		n := convert(fn.pop(), typ)
		val := fn.pop()
		dest := convert(fn.pop(), typ)
		if v, ok := islit(val, "i32"); ok && byte(v) == 0 {
			fn.helpers.add("memory_zero")
			fn.emit(&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: newID("memory_zero"),
					Args: []ast.Expr{
						t.memory.selector,
						dest, n}}})
		} else {
			fn.helpers.add("memory_fill")
			fn.emit(&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: newID("memory_fill"),
					Args: []ast.Expr{
						t.memory.selector,
						dest, val, n}}})
		}

	case 0x0c: // table.init
		elemIdx, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		tableIdx, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		n := fn.pop()
		src := fn.pop()
		dst := fn.pop()
		fn.helpers.add("table_init")
		var tab ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.tables[tableIdx].id}
		if t.tables[tableIdx].imported {
			tab = &ast.StarExpr{X: tab}
		}
		fn.emit(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: newID("table_init"),
				Args: []ast.Expr{
					tab,
					&ast.IndexExpr{
						X:     &ast.SelectorExpr{X: newID("m"), Sel: newID("elements")},
						Index: &ast.BasicLit{Kind: token.INT, Value: formatUint(elemIdx)}},
					dst, src, n}}})

	case 0x0d: // elem.drop
		idx, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		fn.emit(&ast.AssignStmt{
			Tok: token.ASSIGN,
			Lhs: []ast.Expr{&ast.IndexExpr{
				X:     &ast.SelectorExpr{X: newID("m"), Sel: newID("elements")},
				Index: &ast.BasicLit{Kind: token.INT, Value: formatUint(idx)}}},
			Rhs: []ast.Expr{newID("nil")}})

	case 0x0e: // table.copy
		dstIdx, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		srcIdx, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		n := fn.pop()
		src := fn.pop()
		dst := fn.pop()
		fn.helpers.add("table_copy")
		var dstTab ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.tables[dstIdx].id}
		if t.tables[dstIdx].imported {
			dstTab = &ast.StarExpr{X: dstTab}
		}
		var srcTab ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.tables[srcIdx].id}
		if t.tables[srcIdx].imported {
			srcTab = &ast.StarExpr{X: srcTab}
		}
		fn.emit(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun:  newID("table_copy"),
				Args: []ast.Expr{dstTab, srcTab, dst, src, n}}})

	case 0x0f: // table.grow
		idx, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		delta := fn.pop()
		val := fn.pop()
		fn.helpers.add("table_grow")
		var tab ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.tables[idx].id}
		if !t.tables[idx].imported {
			tab = &ast.UnaryExpr{Op: token.AND, X: tab}
		}
		fn.push(&ast.CallExpr{
			Fun: newID("table_grow"),
			Args: []ast.Expr{
				tab, val, delta,
				&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(t.tables[idx].max)}}})

	case 0x10: // table.size
		idx, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		var tab ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.tables[idx].id}
		if t.tables[idx].imported {
			tab = &ast.StarExpr{X: tab}
		}
		fn.pushLazy(convert(&ast.CallExpr{
			Fun:  newID("len"),
			Args: []ast.Expr{tab},
		}, t.tables[idx].stype()))

	case 0x11: // table.fill
		idx, err := readLEB128(t.in)
		if err != nil {
			return err
		}
		n := fn.pop()
		val := fn.pop()
		dest := fn.pop()
		fn.helpers.add("table_fill")
		var tab ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.tables[idx].id}
		if t.tables[idx].imported {
			tab = &ast.StarExpr{X: tab}
		}
		fn.emit(&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun:  newID("table_fill"),
				Args: []ast.Expr{tab, dest, val, n}}})

	case 0x13: // i64.add128
		fn.wideHelper("i64_add128")
	case 0x14: // i64.sub128
		fn.wideHelper("i64_sub128")
	case 0x15: // i64.mul_wide_s
		fn.wideHelper("i64_mul_wide_s")
	case 0x16: // i64.mul_wide_u
		fn.wideHelper("i64_mul_wide_u")

	default:
		return fmt.Errorf("unsupported opcode: 0xFC 0x%02X", code)
	}
	return nil
}

func (fn *funcCompiler) wideHelper(name string) {
	var args []ast.Expr
	if strings.Contains(name, "mul") {
		y := fn.pop()
		x := fn.pop()
		args = []ast.Expr{x, y}
	} else {
		yhi := fn.pop()
		ylo := fn.pop()
		xhi := fn.pop()
		xlo := fn.pop()
		yc, yk := islit(yhi, "i64")
		xc, xk := islit(xhi, "i64")
		if yk && xk && yc == 0 && xc == 0 {
			if name == "i64_add128" {
				name = "i64_add_wide"
			} else {
				name = "i64_sub_wide"
			}
			args = []ast.Expr{xlo, ylo}
		} else {
			args = []ast.Expr{xlo, xhi, ylo, yhi}
		}
	}

	fn.helpers.add(name)
	lo := fn.newTempVal()
	hi := fn.newTempVal()
	fn.emit(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{lo, hi},
		Rhs: []ast.Expr{&ast.CallExpr{Fun: newID(name), Args: args}}})
	fn.pushConst(lo)
	fn.pushConst(hi)
}
