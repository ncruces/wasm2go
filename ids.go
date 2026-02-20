package main

import (
	"go/ast"
	"strconv"
	"strings"
	"unicode"
)

var idCache = map[string]*ast.Ident{}

func newID(name string) *ast.Ident {
	id := idCache[name]
	if id == nil {
		id = ast.NewIdent(name)
		idCache[name] = id
	}
	return id
}

func exportedID(name string) *ast.Ident {
	return ast.NewIdent(identifier("X" + name))
}

func internalID(name string) *ast.Ident {
	return ast.NewIdent(identifier("_" + name))
}

func identifier(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return r
		}
		return '_'
	}, s)
}

func localVar[T interface{ int | uint64 }](i T) *ast.Ident {
	return newID("v" + strconv.Itoa(int(i)))
}

func (fn *funcRef) newTempVar() *ast.Ident {
	id := newID("t" + strconv.Itoa(fn.temps))
	fn.temps++
	return id
}

func (fn *funcRef) newLabel() *ast.Ident {
	lbl := newID("l" + strconv.Itoa(fn.labels))
	fn.labels++
	return lbl
}
