package main

import (
	"encoding/binary"
	"go/ast"
	"go/token"
	"math"
	"strconv"
)

func (t *translator) constI32() (ast.Expr, error) {
	v, err := readSignedLEB128(t.in)
	if err != nil {
		return nil, err
	}

	t.helpers.add("i32")
	// This prevents constant folding/propagation.
	a := []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: formatInt(v)}}
	return &ast.CallExpr{Fun: newID("i32"), Args: a}, nil
}

func (t *translator) constI64() (ast.Expr, error) {
	v, err := readSignedLEB128(t.in)
	if err != nil {
		return nil, err
	}

	t.helpers.add("i64")
	// This prevents constant folding/propagation.
	a := []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: formatInt(v)}}
	return &ast.CallExpr{Fun: newID("i64"), Args: a}, nil
}

func (t *translator) constF32() (ast.Expr, error) {
	var v uint32
	if err := binary.Read(t.in, binary.LittleEndian, &v); err != nil {
		return nil, err
	}

	f := math.Float32frombits(v)
	if -math.MaxFloat32 <= f && f <= +math.MaxFloat32 && (v == 0 || f != 0) {
		a := []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: formatFloat(float64(f), 32)}}
		if (f == 0 || f == 1 || f == -1) && *nanbox {
			t.helpers.add("f32")
			// This prevents constant folding/propagation.
			return &ast.CallExpr{Fun: newID("f32"), Args: a}, nil
		}
		return &ast.CallExpr{Fun: newID("float32"), Args: a}, nil
	}

	// Infinities, NaN, negative zero.
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: newID("math"), Sel: newID("Float32frombits")},
		Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "0x" + strconv.FormatUint(uint64(v), 16)}},
	}, nil
}

func (t *translator) constF64() (ast.Expr, error) {
	var v uint64
	if err := binary.Read(t.in, binary.LittleEndian, &v); err != nil {
		return nil, err
	}

	f := math.Float64frombits(v)
	if -math.MaxFloat64 <= f && f <= +math.MaxFloat64 && (v == 0 || f != 0) {
		a := []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: formatFloat(f, 64)}}
		if (f == 0 || f == 1 || f == -1) && *nanbox {
			t.helpers.add("f64")
			// This prevents constant folding/propagation.
			return &ast.CallExpr{Fun: newID("f64"), Args: a}, nil
		}
		return &ast.CallExpr{Fun: newID("float64"), Args: a}, nil
	}

	// Infinities, NaN, negative zero.
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: newID("math"), Sel: newID("Float64frombits")},
		Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "0x" + strconv.FormatUint(v, 16)}},
	}, nil
}

func (t *translator) globalGet() (ast.Expr, error) {
	v, err := readLEB128(t.in)
	if err != nil {
		return nil, err
	}
	var expr ast.Expr = &ast.SelectorExpr{
		X:   newID("m"),
		Sel: t.globals[v].id,
	}
	if t.globals[v].imported {
		expr = &ast.StarExpr{X: expr}
	}
	return expr, nil
}

func formatInt(i int64) string {
	dec := strconv.FormatInt(i, 10)
	hex := strconv.FormatInt(i, 16)
	if i >= 0 {
		hex = "0x" + hex
	} else {
		hex = "-0x" + hex[1:]
	}
	if complexity(hex) < complexity(dec) {
		return hex
	}
	return dec
}

func formatUint(i uint64) string {
	dec := strconv.FormatUint(i, 10)
	hex := "0x" + strconv.FormatUint(i, 16)
	if complexity(hex) < complexity(dec) {
		return hex
	}
	return dec
}

func formatFloat(f float64, bits int) string {
	dec := strconv.FormatFloat(f, 'g', -1, bits)
	hex := strconv.FormatFloat(f, 'x', -1, bits)
	if complexity(hex) < complexity(dec) {
		return hex
	}
	return dec
}

// This helps decide if a number is better represented
// in decimal or hexadecimal by counting the number of
// of transitions between different characters
// (i.e. ignoring runs of the same character).
// Because s includes the 0x prefix,
// hexadecimal needs a 3 character advantage to win.
func complexity(s string) (transitions int) {
	for i := 1; i < len(s); i++ {
		if s[i] != s[i-1] {
			transitions++
		}
	}
	return transitions
}
