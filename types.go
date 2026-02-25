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

type importKind byte

const (
	functionImport importKind = iota
	tableImport
	memoryImport
	globalImport
)

type importDef struct {
	module string
	name   string
	typ    funcType
}

type tableDef struct {
	id  *ast.Ident
	min int
	max int
}

type memoryDef struct {
	id       *ast.Ident
	imported *ast.Ident
	min      int
	max      int
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

type elemSegment struct {
	init    []uint32
	offset  uint32
	passive bool
}

type dataSegment struct {
	init    []byte
	offset  uint32
	passive bool
}

type nameSubsection byte

const (
	nameModule nameSubsection = iota
	nameFunction
	nameLocal
	nameLabel
	nameType
	nameTable
	nameMemory
	nameGlobal
	nameElem
	nameData
)
