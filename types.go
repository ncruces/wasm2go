package main

import (
	"fmt"
	"go/ast"
)

type wasmType byte

const (
	i32 wasmType = 127 - iota
	i64
	f32
	f64
)

func (t wasmType) Ident() *ast.Ident {
	switch t {
	case i32:
		return newID("int32")
	case i64:
		return newID("int64")
	case f32:
		return newID("float32")
	case f64:
		return newID("float64")
	}
	panic(fmt.Sprintf("unsupported type: %x", byte(t)))
}

type funcType struct {
	params  string // wasmType of parameters
	results string // wasmType of results
}

func (t funcType) toAST() *ast.FuncType {
	return &ast.FuncType{
		Params:  paramsToAST(t.params),
		Results: resultsToAST(t.results),
	}
}

func paramsToAST(types string) *ast.FieldList {
	list := make([]*ast.Field, len(types))
	for i, t := range []byte(types) {
		list[i] = &ast.Field{
			Names: []*ast.Ident{localVar(i)},
			Type:  wasmType(t).Ident(),
		}
	}
	return &ast.FieldList{List: list}
}

func resultsToAST(types string) *ast.FieldList {
	if len(types) == 0 {
		return nil
	}
	list := make([]*ast.Field, len(types))
	for i, t := range []byte(types) {
		list[i] = &ast.Field{Type: wasmType(t).Ident()}
	}
	return &ast.FieldList{List: list}
}

type memoryDef struct {
	id       *ast.Ident
	min, max int
}

type globalDef struct {
	typ  wasmType
	mut  bool
	init ast.Expr
	id   *ast.Ident
}

type exportKind byte

const (
	functionExport exportKind = iota
	tableExport
	memoryExport
	globalExport
)

type export struct {
	kind  exportKind
	index int
}

type dataSegment struct {
	offset int32
	init   []byte
}
