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
	funcref   wasmType = 0x70
	externref wasmType = 0x6f
)

func (t wasmType) ref() bool {
	return t == funcref || t == externref
}

func (t wasmType) ident() *ast.Ident {
	switch t {
	case i32:
		return newID("int32")
	case i64:
		return newID("int64")
	case f32:
		return newID("float32")
	case f64:
		return newID("float64")
	case funcref, externref:
		return newID("any")
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
			Type:  wasmType(t).ident()}
	}
	return &ast.FieldList{List: list}
}

func resultsToAST(types string) *ast.FieldList {
	if len(types) == 0 {
		return nil
	}
	list := make([]*ast.Field, len(types))
	for i, t := range []byte(types) {
		list[i] = &ast.Field{Type: wasmType(t).ident()}
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
	kind   importKind
	fnType funcType
	typ    wasmType
	index  int
}

type tableDef struct {
	id       *ast.Ident
	imported bool
	is64     bool
	min      int
	max      int
}

func (m *tableDef) stype() string {
	if m.is64 {
		return "int64"
	}
	return "int32"
}

type memoryDef struct {
	id       *ast.Ident
	selector ast.Expr
	imported bool
	is64     bool
	min      int64
	max      int64
}

func (m *memoryDef) stype() string {
	if m.is64 {
		return "int64"
	}
	return "int32"
}

func (m *memoryDef) utype() string {
	return "u" + m.stype()
}

type globalDef struct {
	id       *ast.Ident
	typ      wasmType
	imported bool
	init     ast.Expr
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
	index   uint32
	offset  uint32
	passive bool
}

type dataSegment struct {
	init    []byte
	offset  uint64
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
