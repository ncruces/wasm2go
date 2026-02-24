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
	t.helpers.add("i32_const")
	return &ast.CallExpr{
		Fun:  newID("i32_const"),
		Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(v, 10)}},
	}, nil
}

func (t *translator) constI64() (ast.Expr, error) {
	v, err := readSignedLEB128(t.in)
	if err != nil {
		return nil, err
	}
	t.helpers.add("i64_const")
	return &ast.CallExpr{
		Fun:  newID("i64_const"),
		Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(v, 10)}},
	}, nil
}

func (t *translator) constF32() (ast.Expr, error) {
	var v uint32
	if err := binary.Read(t.in, binary.LittleEndian, &v); err != nil {
		return nil, err
	}

	f := math.Float32frombits(v)
	if -math.MaxFloat32 <= f && f <= +math.MaxFloat32 {
		return &ast.CallExpr{
			Fun:  newID("float32"),
			Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: strconv.FormatFloat(float64(f), 'g', -1, 32)}},
		}, nil
	}

	t.packages.add("math")
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: newID("math"), Sel: newID("Float32frombits")},
		Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(uint64(v), 10)}},
	}, nil
}

func (t *translator) constF64() (ast.Expr, error) {
	var v uint64
	if err := binary.Read(t.in, binary.LittleEndian, &v); err != nil {
		return nil, err
	}

	f := math.Float64frombits(v)
	if -math.MaxFloat64 <= f && f <= +math.MaxFloat64 {
		return &ast.CallExpr{
			Fun:  newID("float64"),
			Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: strconv.FormatFloat(f, 'g', -1, 64)}},
		}, nil
	}

	t.packages.add("math")
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: newID("math"), Sel: newID("Float64frombits")},
		Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(v, 10)}},
	}, nil
}

func (t *translator) globalGet() (ast.Expr, error) {
	v, err := readLEB128(t.in)
	if err != nil {
		return nil, err
	}
	return &ast.SelectorExpr{
		X:   newID("m"),
		Sel: t.globals[v].id,
	}, nil
}
