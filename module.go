package main

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
)

func (t *translator) createModuleStruct(hostInterfaces []*ast.GenDecl) (*ast.GenDecl, *ast.TypeSpec) {
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
			id := newID("Memory")
			fields = append(fields, &ast.Field{
				Names: []*ast.Ident{t.memory.id},
				Type:  &ast.StarExpr{X: &ast.ArrayType{Elt: newID("byte")}},
			}, &ast.Field{Names: []*ast.Ident{id}, Type: id})
		} else {
			fields = append(fields, &ast.Field{
				Names: []*ast.Ident{t.memory.id},
				Type:  &ast.ArrayType{Elt: newID("byte")}})
		}
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{newID("maxMem")},
			Type:  newID("int64")})
	}
	// Globals: owned are type; imported *type.
	for _, g := range t.globals {
		var typ ast.Expr = g.typ.ident()
		if g.imported {
			typ = &ast.StarExpr{X: typ}
		}
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{g.id},
			Type:  typ})
	}
	// Imported modules.
	seen := set[string]{}
	for _, imp := range t.imports {
		if !seen.has(imp.module) {
			seen.add(imp.module)
			fields = append(fields, &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(internal(imp.module))},
				Type:  ast.NewIdent(exported(imp.module))})
		}
	}
	for _, g := range t.globals {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{g.id},
			Type:  g.typ.Ident()})
	}
	var typeParams *ast.FieldList
	if len(hostInterfaces) > 0 {
		typeParams = new(ast.FieldList)
		for _, decl := range hostInterfaces {
			typeSpec := decl.Specs[0].(*ast.TypeSpec)
			typeParams.List = append(typeParams.List, &ast.Field{
				Names: []*ast.Ident{typeSpec.Name},
				Type:  newID(typeSpec.Name.Name),
			})
		}
	}
	typeSpec := &ast.TypeSpec{
		Name:       newID("Module"),
		TypeParams: typeParams,
		Type: &ast.StructType{
			Fields: &ast.FieldList{List: fields},
		},
	}
	return &ast.GenDecl{
		Tok:   token.TYPE,
		Specs: []ast.Spec{typeSpec},
	}, typeSpec
}

func (t *translator) createHostInterfaces() []*ast.GenDecl {
	ifaces := map[string][]*ast.Field{}

	for _, imp := range t.imports {
		typ := imp.typ.toAST()

		ifaces[imp.module] = append(ifaces[imp.module], &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(exported(imp.name))},
			Type:  typ})
	}

	decls := make([]*ast.GenDecl, 0, len(ifaces))
	for name, methods := range ifaces {
		decls = append(decls, &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{&ast.TypeSpec{
				Name: ast.NewIdent(exported(name)),
				Type: &ast.InterfaceType{Methods: &ast.FieldList{List: methods}}}}})
	}
	return decls
}

func (t *translator) createNewFunc() ast.Decl {
	// Create a new instance of module.
	body := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{newID("m")},
				Rhs: []ast.Expr{&ast.UnaryExpr{
					Op: token.AND,
					X:  &ast.CompositeLit{Type: newID("Module")}}}}},
	}

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
				Type:  ast.NewIdent(exported(imp.module))})
			body.List = append(body.List, &ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{&ast.SelectorExpr{
					X:   newID("m"),
					Sel: ast.NewIdent(internal(imp.module))}},
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
				&ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(uint64(t.memory.max), 10)}}})
		if !t.memory.imported {
			body.List = append(body.List, &ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{t.memory.selector},
				Rhs: []ast.Expr{&ast.CallExpr{
					Fun: newID("make"),
					Args: []ast.Expr{
						&ast.ArrayType{Elt: newID("byte")},
						&ast.BasicLit{
							Kind:  token.INT,
							Value: strconv.FormatUint(uint64(t.memory.min)<<16, 10)}}}}})
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
						Sel: ast.NewIdent(exported(imp.name))}}}})

		case externGlobal:
			body.List = append(body.List, &ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: t.globals[imp.index].id}},
				Rhs: []ast.Expr{&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   locals[imp.module],
						Sel: ast.NewIdent(exported(imp.name))}}}})

		case externMemory:
			body.List = append(body.List, &ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: newID("Memory")}},
				Rhs: []ast.Expr{&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   locals[imp.module],
						Sel: ast.NewIdent(exported(imp.name))}}},
			}, &ast.AssignStmt{
				Tok: token.ASSIGN,
				Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: t.memory.id}},
				Rhs: []ast.Expr{&ast.CallExpr{
					Fun: &ast.SelectorExpr{X: &ast.SelectorExpr{X: newID("m"), Sel: newID("Memory")}, Sel: newID("Slice")}}}})
		}
	}

	// Intialize the tables.
	if len(t.elements) > 0 {
		elts := make([]ast.Expr, len(t.elements))
		for i, elem := range t.elements {
			inner := make([]ast.Expr, len(elem.init))
			for j, idx := range elem.init {
				inner[j] = &ast.SelectorExpr{X: newID("m"), Sel: t.functions[idx].decl.Name}
			}
			elts[i] = &ast.CompositeLit{Elts: inner}
		}
		body.List = append(body.List, &ast.AssignStmt{
			Tok: token.ASSIGN,
			Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: newID("elements")}},
			Rhs: []ast.Expr{&ast.CompositeLit{
				Type: &ast.ArrayType{Elt: &ast.ArrayType{Elt: newID("any")}},
				Elts: elts}}})

		for i, elem := range t.elements {
			if elem.passive {
				continue
			}
			var tab ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: t.tables[elem.index].id}
			if t.tables[elem.index].imported {
				tab = &ast.StarExpr{X: tab}
			}
			body.List = append(body.List, &ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: newID("copy"),
					Args: []ast.Expr{
						&ast.SliceExpr{
							X:   tab,
							Low: &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(int(elem.offset))}},
						&ast.IndexExpr{
							X:     &ast.SelectorExpr{X: newID("m"), Sel: newID("elements")},
							Index: &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(i)}}}}})
		}
	}
	// Intialize the memory.
	for i, seg := range t.data {
		if seg.passive {
			continue
		}
		body.List = append(body.List, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: newID("copy"),
				Args: []ast.Expr{
					&ast.SliceExpr{
						X:   t.memory.selector,
						Low: &ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(seg.offset, 10)}},
					dataID(i)}}})
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
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   newID("m"),
					Sel: t.functions[^t.start].decl.Name}}})
	}

	body.List = append(body.List, &ast.ReturnStmt{Results: []ast.Expr{newID("m")}})

	return &ast.FuncDecl{
		Name: newID("New"),
		Type: &ast.FuncType{
			TypeParams: t.moduleType.TypeParams,
			Params:     &ast.FieldList{List: params},
			Results:    &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: newID("Module")}}}},
		},
		Body: body,
	}
}

func (t *translator) createHostInterfaces() []ast.Decl {
	ifaces := map[string][]*ast.Field{}

	for _, imp := range t.imports {
		var typ ast.Expr
		switch imp.kind {
		case externFunction:
			typ = imp.fnType.toAST()
		case externTable: // *[]any
			typ = &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.ArrayType{Elt: newID("any")}}}}}}
		case externGlobal: // *type
			typ = &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: imp.typ.ident()}}}}}
		case externMemory: // Memory
			typ = &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: newID("Memory")}}}}
		}

		ifaces[imp.module] = append(ifaces[imp.module], &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(exported(imp.name))},
			Type:  typ})
	}

	decls := make([]ast.Decl, 0, len(ifaces))
	for name, methods := range ifaces {
		decls = append(decls, &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{&ast.TypeSpec{
				Assign: 1,
				Name:   ast.NewIdent(exported(name)),
				Type:   &ast.InterfaceType{Methods: &ast.FieldList{List: methods}}}}})
	}
	return decls
}

func (t *translator) createMemoryTypes() []ast.Decl {
	var decls []ast.Decl
	// Memory interface as a type alias.
	if !*nohost {
		decls = append(decls, &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{&ast.TypeSpec{
				Assign: 1,
				Name:   newID("Memory"),
				Type: &ast.InterfaceType{Methods: &ast.FieldList{
					List: []*ast.Field{{
						Names: []*ast.Ident{newID("Slice")},
						Type: &ast.FuncType{
							Results: &ast.FieldList{
								List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.ArrayType{Elt: newID("byte")}}}}}},
					}, {
						Names: []*ast.Ident{newID("Grow")},
						Type: &ast.FuncType{
							Params: &ast.FieldList{
								List: []*ast.Field{{
									Names: []*ast.Ident{newID("delta"), newID("max")},
									Type:  newID("int64")}}},
							Results: &ast.FieldList{
								List: []*ast.Field{{Type: newID("int64")}}}}}}}}}}})
	}
	// Memory structure implementing the interface for owned memory.
	if !t.memory.imported {
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
				Fun:  &ast.ParenExpr{X: &ast.StarExpr{X: &ast.ArrayType{Elt: newID("byte")}}},
				Args: []ast.Expr{newID("m")}}}}}},
		}, &ast.FuncDecl{
			Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{newID("m")}, Type: &ast.StarExpr{X: newID("wasmMemory")}}}},
			Name: newID("Grow"),
			Type: &ast.FuncType{
				Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{newID("delta"), newID("max")}, Type: newID("int64")}}},
				Results: &ast.FieldList{List: []*ast.Field{{Type: newID("int64")}}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{
				Fun: newID("memory_grow"),
				Args: []ast.Expr{
					&ast.CallExpr{
						Fun:  &ast.ParenExpr{X: &ast.StarExpr{X: &ast.ArrayType{Elt: newID("byte")}}},
						Args: []ast.Expr{newID("m")}},
					newID("delta"),
					newID("max")}}}}}}})
		t.helpers.add("memory_grow")
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
		methodName := ast.NewIdent(exported(name))

		var results *ast.FieldList
		var body []ast.Stmt

		switch exp.kind {
		case externFunction:
			continue
		case externTable:
			tab := t.tables[exp.index]
			results = &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.ArrayType{Elt: newID("any")}}}}}
			var ret ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: tab.id}
			if !tab.imported {
				ret = &ast.UnaryExpr{Op: token.AND, X: ret}
			}
			body = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ret}}}
		case externGlobal:
			g := t.globals[exp.index]
			results = &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: g.typ.ident()}}}}
			var ret ast.Expr = &ast.SelectorExpr{X: newID("m"), Sel: g.id}
			if !g.imported {
				ret = &ast.UnaryExpr{Op: token.AND, X: ret}
			}
			body = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ret}}}
		case externMemory:
			results = &ast.FieldList{List: []*ast.Field{{Type: newID("Memory")}}}
			if t.memory.imported {
				body = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: newID("Memory")}}}}
			} else {
				body = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{
					Fun:  &ast.ParenExpr{X: &ast.StarExpr{X: newID("wasmMemory")}},
					Args: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: &ast.SelectorExpr{X: newID("m"), Sel: t.memory.id}}}}}}}
			}
		}

		decls = append(decls, &ast.FuncDecl{
			Recv: modRecvList,
			Name: methodName,
			Type: &ast.FuncType{Results: results},
			Body: &ast.BlockStmt{List: body}})
	}
	return decls
}
