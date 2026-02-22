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

func exported(name string) string {
	var buf strings.Builder
	for i, r := range name {
		if i == 0 {
			if u := unicode.ToUpper(r); unicode.IsUpper(u) {
				buf.WriteRune(u)
				continue
			}
			buf.WriteByte('X')
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			r = '_'
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

func internal(name string) string {
	var buf strings.Builder
	for i, r := range name {
		if i == 0 && unicode.IsUpper(r) {
			buf.WriteByte('_')
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			r = '_'
		}
		buf.WriteRune(r)
	}
	return buf.String()
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
