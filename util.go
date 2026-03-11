package main

import "go/ast"

// appendDecl uses generic parameter typing to allow typical variable argument
// passing of ast.Decl implementations to append to an []ast.Decl slice.
func appendDecl[D ast.Decl](decls []ast.Decl, decl ...D) []ast.Decl {
	for _, decl := range decl {
		decls = append(decls, decl)
	}
	return decls
}
