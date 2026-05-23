package main

import (
	"go/ast"
	"strconv"
)

// Names in use by the translator (consider for collisions).
//
// At top level we may have:
//	- a Module structure
//	- a New function
//	- a Memory interface alias
//	- a wasmMemory type
//	- data constants (data+number)
//	- interface aliases for imports (X-prefixed)
//  - internal module functions (fn+number or _-prefixed)
//	- helper functions (incl. i32/i64/f32/f64)
//
// Module members may have:
//	- tables (t+number or _-prefixed)
//	- globals (g+number or _-prefixed)
//	- functions (fn+number or _-prefixed)
//	- exports (X-prefixed)
//	- elements
//	- memory
//
// Function code may have:
//	- local variables, arguments (v+number)
//	- materialized temporaries (t+number)
//	- temporary variables (p+number)
//	- labels (l+number)

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
