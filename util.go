package main

import "go/ast"

func toDecl[D ast.Decl](decl ...D) []ast.Decl {
	decls := make([]ast.Decl, 0, len(decl))
	for _, d := range decl {
		decls = append(decls, d)
	}
	return decls
}
