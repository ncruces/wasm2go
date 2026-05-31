package main

import (
	"go/ast"
	"go/token"
	"maps"
	"slices"
	"strconv"

	"github.com/ncruces/wasm2go/internal/mangle"
)

var modRecvList = &ast.FieldList{List: []*ast.Field{{
	Names: []*ast.Ident{newID("m")},
	Type:  &ast.StarExpr{X: newID("Module")}}}}

func (t *translator) createModuleStruct() ast.Decl {
	var fields []*ast.Field
	// Tables: owned are []any; imported *[]any.
	for _, tab := range t.tables {
		var typ ast.Expr = &ast.ArrayType{Elt: newID("any")}
		if tab.imported {
			typ = &ast.StarExpr{X: typ}
		}
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{tab.id},
			Type:  typ})
	}
	// Elements: table initializers, [][]any.
	if len(t.elements) > 0 {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{newID("elements")},
			Type:  &ast.ArrayType{Elt: &ast.ArrayType{Elt: newID("any")}}})
	}
	// Memory: owned a []byte; imported a *[]byte and a Memory field.
	if t.memory != nil {
		if t.memory.imported {
			fields = append(fields, &ast.Field{
				Names: []*ast.Ident{t.memory.id},
				Type:  &ast.StarExpr{X: &ast.ArrayType{Elt: newID("byte")}},
			}, &ast.Field{Names: []*ast.Ident{newID("memImp")}, Type: newID("Memory")})
		} else {
			fields = append(fields, &ast.Field{
				Names: []*ast.Ident{t.memory.id},
				Type:  &ast.ArrayType{Elt: newID("byte")}})
		}
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{newID("maxMem")},
			Type:  newID("int64")})
		if t.memory.shared || t.helpers.has("atomic_waiters") {
			fields = append(fields, &ast.Field{
				Names: []*ast.Ident{newID("waiters")},
				Type:  &ast.StarExpr{X: &ast.SelectorExpr{X: newID("sync"), Sel: newID("Map")}}})
			t.packages.add("sync")
		}
	}
	// Globals: owned/immutable are type; imported *type.
	for _, g := range t.globals {
		var typ ast.Expr = g.typ.ident()
		if g.imported && g.mutable {
			typ = &ast.StarExpr{X: typ}
		}
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{g.id},
			Type:  typ})
	}
	// Imported modules.
	seen := set[string]{}
	for _, imp := range t.imports {
		if seen.add(imp.module) {
			fields = append(fields, &ast.Field{
				Names: []*ast.Ident{mangle.ID(imp.module, mangle.Internal)},
				Type:  mangle.ID(imp.module, mangle.Exported)})
		}
	}

	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: newID("Module"),
				Type: &ast.StructType{Fields: &ast.FieldList{List: fields}}}},
	}
}

func (t *translator) createNewFunc() ast.Decl {
	// Create a new instance of module.
	body := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{newID("m")},
				Rhs: []ast.Expr{&ast.CallExpr{
					Fun:  newID("new"),
					Args: []ast.Expr{newID("Module")}}}}}}

	// Create the params, and initialize fields.
	params := []*ast.Field{}
	locals := map[string]*ast.Ident{}
	{
		var i int
		for _, imp := range t.imports {
			if locals[imp.module] != nil {
				continue
			}
			local := localVar(i)
			locals[imp.module] = local
			i++

			params = append(params, &ast.Field{
				Names: []*ast.Ident{local},
				Type:  mangle.ID(imp.module, mangle.Exported)})
			body.List = append(body.List, &ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{&ast.SelectorExpr{
					X:   newID("m"),
					Sel: mangle.ID(imp.module, mangle.Internal)}},
				Rhs: []ast.Expr{local}})
		}
	}

	// Create owned tables.
	for _, tab := range t.tables {
		if tab.imported || tab.min == 0 {
			continue
		}
		body.List = append(body.List, &ast.AssignStmt{
			Tok: token.ASSIGN,
			Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: tab.id}},
			Rhs: []ast.Expr{&ast.CallExpr{
				Fun: newID("make"),
				Args: []ast.Expr{
					&ast.ArrayType{Elt: newID("any")},
					&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(tab.min)}}}}})
	}
	// Create owned memory.
	if t.memory != nil {
		body.List = append(body.List, &ast.AssignStmt{
			Tok: token.ASSIGN,
			Lhs: []ast.Expr{&ast.SelectorExpr{
				X:   newID("m"),
				Sel: newID("maxMem")}},
			Rhs: []ast.Expr{
				&ast.BasicLit{Kind: token.INT, Value: formatInt(t.memory.max)}}})
		if !t.memory.imported {
			args := []ast.Expr{
				&ast.ArrayType{Elt: newID("byte")},
				&ast.BasicLit{Kind: token.INT, Value: formatInt(t.memory.min << 16)},
			}
			if t.memory.shared {
				args = append(args, &ast.BasicLit{Kind: token.INT, Value: formatInt(t.memory.max << 16)})
			}
			body.List = append(body.List, &ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{t.memory.selector},
				Rhs: []ast.Expr{&ast.CallExpr{Fun: newID("make"), Args: args}}})
			if t.memory.shared {
				body.List = append(body.List, &ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: newID("waiters")}},
					Rhs: []ast.Expr{&ast.CallExpr{
						Fun:  newID("new"),
						Args: []ast.Expr{&ast.SelectorExpr{X: newID("sync"), Sel: newID("Map")}}}}})
			}
		}
	}

	// Set imported tables, globals and memory.
	for _, imp := range t.imports {
		switch imp.kind {
		case externTable:
			body.List = append(body.List, &ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: t.tables[imp.index].id}},
				Rhs: []ast.Expr{&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   locals[imp.module],
						Sel: mangle.ID(imp.name, mangle.Exported)}}}})

		case externGlobal:
			var rhs ast.Expr = &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   locals[imp.module],
					Sel: mangle.ID(imp.name, mangle.Exported)}}
			if !t.globals[imp.index].mutable {
				rhs = &ast.StarExpr{X: rhs}
			}
			body.List = append(body.List, &ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: t.globals[imp.index].id}},
				Rhs: []ast.Expr{rhs}})

		case externMemory:
			expr := &ast.SelectorExpr{X: newID("m"), Sel: newID("memImp")}
			body.List = append(body.List, &ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{expr},
				Rhs: []ast.Expr{&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   locals[imp.module],
						Sel: mangle.ID(imp.name, mangle.Exported)}}},
			}, &ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: t.memory.id}},
				Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.SelectorExpr{X: expr, Sel: newID("Slice")}}}})

			if t.memory.shared {
				body.List = append(body.List, &ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: newID("waiters")}},
					Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.SelectorExpr{X: expr, Sel: newID("Waiters")}}}})
			}
		}
	}

	// Intialize the tables.
	if len(t.elements) > 0 {
		elts := make([]ast.Expr, len(t.elements))
		for i, elem := range t.elements {
			if elem.declive {
				elts[i] = newID("nil")
			} else {
				elts[i] = &ast.CompositeLit{Elts: elem.init}
			}
		}
		body.List = append(body.List, &ast.AssignStmt{
			Tok: token.ASSIGN,
			Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: newID("elements")}},
			Rhs: []ast.Expr{&ast.CompositeLit{
				Type: &ast.ArrayType{Elt: &ast.ArrayType{Elt: newID("any")}},
				Elts: elts}}})

		for i, elem := range t.elements {
			if elem.passive || elem.declive {
				continue
			}
			var tab ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.tables[elem.index].id}
			if t.tables[elem.index].imported {
				tab = &ast.StarExpr{X: tab}
			}
			body.List = append(body.List,
				&ast.ExprStmt{X: &ast.CallExpr{
					Fun: newID("copy"),
					Args: []ast.Expr{
						&ast.SliceExpr{X: tab, Low: elem.offset},
						&ast.IndexExpr{
							X:     &ast.SelectorExpr{X: newID("m"), Sel: newID("elements")},
							Index: &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(i)}}}}},
				&ast.AssignStmt{
					Tok: token.ASSIGN,
					Lhs: []ast.Expr{&ast.IndexExpr{
						X:     &ast.SelectorExpr{X: newID("m"), Sel: newID("elements")},
						Index: &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(i)}}},
					Rhs: []ast.Expr{newID("nil")}})
		}
	}
	// Intialize the memory.
	for i, seg := range t.data {
		if seg.passive || seg.merged {
			continue
		}
		body.List = append(body.List, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: newID("copy"),
				Args: []ast.Expr{
					&ast.SliceExpr{
						X:   t.memory.selector,
						Low: convert(seg.offset, t.memory.utype())},
					t.dataExpr(i)}}})
	}
	// Create and initialize owned globals.
	for _, g := range t.globals {
		if g.imported {
			continue
		}
		body.List = append(body.List, &ast.AssignStmt{
			Tok: token.ASSIGN,
			Lhs: []ast.Expr{&ast.SelectorExpr{
				X:   newID("m"),
				Sel: g.id,
			}},
			Rhs: []ast.Expr{g.init}})
	}

	// Initialize imported modules.
	for _, param := range params {
		body.List = append(body.List, &ast.IfStmt{
			Init: &ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{newID("i"), newID("ok")},
				Rhs: []ast.Expr{&ast.TypeAssertExpr{
					X: &ast.CallExpr{Fun: newID("any"), Args: []ast.Expr{param.Names[0]}},
					Type: &ast.InterfaceType{
						Methods: &ast.FieldList{List: []*ast.Field{{
							Names: []*ast.Ident{newID("Init")},
							Type: &ast.FuncType{
								Params: &ast.FieldList{List: []*ast.Field{{
									Type: newID("any")}}}}}}}}}}},
			Cond: newID("ok"),
			Body: &ast.BlockStmt{
				List: []ast.Stmt{&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun:  &ast.SelectorExpr{X: newID("i"), Sel: newID("Init")},
						Args: []ast.Expr{newID("m")}}}}}})
	}
	// Call start function.
	if t.start != 0 {
		body.List = append(body.List, &ast.ExprStmt{
			X: &ast.CallExpr{Fun: t.functions[^t.start].call}})
	}

	body.List = append(body.List, &ast.ReturnStmt{Results: []ast.Expr{newID("m")}})

	return &ast.FuncDecl{
		Name: newID("New"),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: params},
			Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: newID("Module")}}}}},
		Body: body}
}

func (t *translator) createHostInterfaces() []ast.Decl {
	ifaces := map[string][]*ast.Field{}

	seen := set[struct{ module, name string }]{}
	for _, imp := range t.imports {
		if !seen.add(struct{ module, name string }{imp.module, imp.name}) {
			continue
		}

		var typ ast.Expr
		switch imp.kind {
		case externFunction:
			typ = imp.fnType.toAST(true)
		case externTable: // *[]any
			typ = &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.ArrayType{Elt: newID("any")}}}}}}
		case externGlobal: // *type
			typ = &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: imp.typ.ident()}}}}}
		case externMemory: // Memory
			typ = &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: newID("Memory")}}}}
		}

		ifaces[imp.module] = append(ifaces[imp.module], &ast.Field{
			Names: []*ast.Ident{mangle.ID(imp.name, mangle.Exported)},
			Type:  typ})
	}

	decls := make([]ast.Decl, 0, len(ifaces))
	for _, name := range slices.Sorted(maps.Keys(ifaces)) {
		methods := ifaces[name]
		decls = append(decls, &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{&ast.TypeSpec{
				Assign: 1,
				Name:   mangle.ID(name, mangle.Exported),
				Type:   &ast.InterfaceType{Methods: &ast.FieldList{List: methods}}}}})
	}
	return decls
}

func (t *translator) createMemoryTypes() []ast.Decl {
	var decls []ast.Decl
	// Memory interface as a type alias.
	if !*nohost {
		iface := &ast.InterfaceType{Methods: &ast.FieldList{
			List: []*ast.Field{{
				Names: []*ast.Ident{newID("Slice")},
				Type: &ast.FuncType{
					Results: &ast.FieldList{
						List: []*ast.Field{{Type: &ast.StarExpr{
							X: &ast.ArrayType{Elt: newID("byte")}}}}}},
			}, {
				Names: []*ast.Ident{newID("Grow")},
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{{
							Names: []*ast.Ident{newID("delta"), newID("max")},
							Type:  newID("int64")}}},
					Results: &ast.FieldList{
						List: []*ast.Field{{Type: newID("int64")}}}}}}}}

		if t.memory.shared {
			iface.Methods.List = append(iface.Methods.List, &ast.Field{
				Names: []*ast.Ident{newID("Waiters")},
				Type: &ast.FuncType{
					Results: &ast.FieldList{
						List: []*ast.Field{{Type: &ast.StarExpr{
							X: &ast.SelectorExpr{X: newID("sync"), Sel: newID("Map")}}}}}}})
		}

		decls = append(decls,
			&ast.GenDecl{
				Tok: token.TYPE,
				Specs: []ast.Spec{&ast.TypeSpec{
					Assign: 1, Name: newID("Memory"), Type: iface}}})
	}
	// Memory structure implementing the interface for owned memory.
	if !t.memory.imported {
		name := "memory_grow"
		if t.memory.shared {
			name = "atomic_memory_grow"
			needsUnsafe("shared memory")
		}

		decls = append(decls, &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{&ast.TypeSpec{
				Name: newID("wasmMemory"),
				Type: &ast.ArrayType{Elt: newID("byte")}}},
		}, &ast.FuncDecl{
			Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{newID("m")}, Type: &ast.StarExpr{X: newID("wasmMemory")}}}},
			Name: newID("Slice"),
			Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.ArrayType{Elt: newID("byte")}}}}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{
				Fun:  &ast.StarExpr{X: &ast.ArrayType{Elt: newID("byte")}},
				Args: []ast.Expr{newID("m")}}}}}},
		}, &ast.FuncDecl{
			Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{newID("m")}, Type: &ast.StarExpr{X: newID("wasmMemory")}}}},
			Name: newID("Grow"),
			Type: &ast.FuncType{
				Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{newID("delta"), newID("max")}, Type: newID("int64")}}},
				Results: &ast.FieldList{List: []*ast.Field{{Type: newID("int64")}}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{
				Fun: newID(name),
				Args: []ast.Expr{
					&ast.CallExpr{
						Fun:  &ast.StarExpr{X: &ast.ArrayType{Elt: newID("byte")}},
						Args: []ast.Expr{newID("m")}},
					newID("delta"),
					newID("max")}}}}}}})
		t.helpers.add(name)

		if t.memory.shared {
			decls = append(decls, &ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{newID("m")}, Type: &ast.StarExpr{X: newID("wasmMemory")}}}},
				Name: newID("Waiters"),
				Type: &ast.FuncType{
					Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.SelectorExpr{X: newID("sync"), Sel: newID("Map")}}}}}},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{
					&ast.SelectorExpr{X: newID("m"), Sel: newID("waiters")}}}}}})
		}
	}
	return decls
}

func (t *translator) createExportMethods() []ast.Decl {
	var decls []ast.Decl

	names := make([]string, 0, len(t.exports))
	for name := range t.exports {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		exp := t.exports[name]
		name := mangle.Name(name, mangle.Exported)

		var decl = &ast.FuncDecl{
			Recv: modRecvList,
			Name: ast.NewIdent(name),
			Type: &ast.FuncType{},
			Body: &ast.BlockStmt{}}

		switch exp.kind {
		case externFunction:
			fn := t.functions[exp.index]
			if name == fn.decl.Name.Name {
				continue
			}

			decl.Type = fn.typ.toAST(true)
			call := &ast.CallExpr{Fun: fn.call}
			for i := range fn.typ.params {
				call.Args = append(call.Args, localVar(i))
			}
			if len(fn.typ.results) == 0 {
				decl.Body.List = []ast.Stmt{&ast.ExprStmt{X: call}}
			} else {
				decl.Body.List = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}
			}
		case externTable:
			tab := t.tables[exp.index]
			decl.Type.Results = &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.ArrayType{Elt: newID("any")}}}}}
			var ret ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: tab.id}
			if !tab.imported {
				ret = &ast.UnaryExpr{Op: token.AND, X: ret}
			}
			decl.Body.List = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ret}}}
		case externGlobal:
			g := t.globals[exp.index]
			decl.Type.Results = &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: g.typ.ident()}}}}
			var ret ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: g.id}
			if !(g.imported && g.mutable) {
				ret = &ast.UnaryExpr{Op: token.AND, X: ret}
			}
			decl.Body.List = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ret}}}
		case externMemory:
			decl.Type.Results = &ast.FieldList{List: []*ast.Field{{Type: newID("Memory")}}}
			if t.memory.imported {
				decl.Body.List = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: newID("memImp")}}}}
			} else {
				decl.Body.List = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{
					Fun:  &ast.ParenExpr{X: &ast.StarExpr{X: newID("wasmMemory")}},
					Args: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: &ast.SelectorExpr{X: newID("m"), Sel: t.memory.id}}}}}}}
			}
		}

		decls = append(decls, decl)
	}
	return decls
}

func (t *translator) createDylinkConstants() ast.Decl {
	return &ast.FuncDecl{
		Name: newID("DylinkInfo"),
		Type: &ast.FuncType{
			Results: &ast.FieldList{
				List: []*ast.Field{{
					Names: []*ast.Ident{
						newID("memorySize"),
						newID("memoryAlignment"),
						newID("tableSize"),
						newID("tableAlignment")},
					Type: newID("int64")}}}},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.BasicLit{Kind: token.INT, Value: formatInt(t.dylink.memorySize)},
					&ast.BasicLit{Kind: token.INT, Value: formatInt(1 << t.dylink.memoryAlignment)},
					&ast.BasicLit{Kind: token.INT, Value: formatInt(t.dylink.tableSize)},
					&ast.BasicLit{Kind: token.INT, Value: formatInt(1 << t.dylink.tableAlignment)}}}}}}
}
