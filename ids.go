package main

import (
	"go/ast"
	"strconv"
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

func dataID[T interface{ int | uint64 }](i T) *ast.Ident {
	return newID("data" + strconv.Itoa(int(i)))
}

func localVar[T interface{ int | uint64 }](i T) *ast.Ident {
	return newID("v" + strconv.Itoa(int(i)))
}

func tempVal(i int) *ast.Ident {
	return newID("t" + strconv.Itoa(i))
}

func tempVar(i int) *ast.Ident {
	return newID("p" + strconv.Itoa(i))
}

func labelId(i int) *ast.Ident {
	// Labels can be relabeled, merged; don't cache!
	return ast.NewIdent("l" + strconv.Itoa(i))
}
