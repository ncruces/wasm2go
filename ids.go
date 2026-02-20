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

func tempVar(i int) *ast.Ident {
	return newID("t" + strconv.Itoa(i))
}

func labelId(i int) *ast.Ident {
	return newID("l" + strconv.Itoa(i))
}
