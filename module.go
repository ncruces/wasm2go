package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

var modRecvList = &ast.FieldList{List: []*ast.Field{{
	Names: []*ast.Ident{newID("m")},
	Type:  newID("Module"),
}}}

func (t *translator) createModuleStruct() ast.Decl {
	var fields []*ast.Field
	if t.table != nil {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{t.table.id},
			Type:  &ast.ArrayType{Elt: newID("any")},
		})
	}
	if len(t.elements) > 0 {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{newID("elements")},
			Type:  &ast.ArrayType{Elt: &ast.ArrayType{Elt: newID("any")}},
		})
	}
	if t.memory != nil {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{t.memory.id},
			Type:  &ast.ArrayType{Elt: newID("byte")},
		})
	}
	for _, g := range t.globals {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{g.id},
			Type:  g.typ.Ident(),
		})
	}
	seen := set[string]{}
	for _, imp := range t.imports {
		if !seen.has(imp.module) {
			seen.add(imp.module)
			fields = append(fields, &ast.Field{
				Names: []*ast.Ident{newID(internal(imp.module))},
				Type:  newID(imported(imp.module)),
			})
		}
	}
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: newID("Module"),
				Type: &ast.StructType{
					Fields: &ast.FieldList{List: fields},
				},
			},
		},
	}
}

func (t *translator) createHostInterfaces() []ast.Decl {
	ifaces := map[string][]*ast.Field{}

	for _, imp := range t.imports {
		typ := imp.typ.toAST()
		typ.Params.List = append([]*ast.Field{{
			Names: []*ast.Ident{newID("m")},
			Type:  &ast.StarExpr{X: newID("Module")},
		}}, typ.Params.List...)

		ifaces[imp.module] = append(ifaces[imp.module], &ast.Field{
			Names: []*ast.Ident{newID(imported(imp.name))},
			Type:  typ,
		})
	}

	decls := make([]ast.Decl, 0, len(ifaces))
	for name, methods := range ifaces {
		decls = append(decls, &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{&ast.TypeSpec{
				Name: newID(imported(name)),
				Type: &ast.InterfaceType{Methods: &ast.FieldList{List: methods}},
			}},
		})
	}
	return decls
}

func (t *translator) createNewFunc() ast.Decl {
	var params []*ast.Field
	body := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{newID("m")},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{&ast.UnaryExpr{
					Op: token.AND,
					X:  &ast.CompositeLit{Type: newID("Module")},
				}},
			},
		},
	}

	seen := set[string]{}
	for i, imp := range t.imports {
		if !seen.has(imp.module) {
			seen.add(imp.module)
			params = append(params, &ast.Field{
				Names: []*ast.Ident{localVar(i)},
				Type:  newID(imported(imp.module)),
			})
			body.List = append(body.List, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.SelectorExpr{
					X:   newID("m"),
					Sel: newID(internal(imp.module)),
				}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{localVar(i)},
			})
		}
	}

	if t.table != nil {
		body.List = append(body.List, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.SelectorExpr{
				X:   newID("m"),
				Sel: t.table.id,
			}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.CallExpr{
				Fun: newID("make"),
				Args: []ast.Expr{
					&ast.ArrayType{Elt: newID("any")},
					&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(t.table.min)},
				},
			}},
		})
	}

	if t.memory != nil {
		body.List = append(body.List, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.SelectorExpr{
				X:   newID("m"),
				Sel: t.memory.id,
			}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.CallExpr{
				Fun: newID("make"),
				Args: []ast.Expr{
					&ast.ArrayType{Elt: newID("byte")},
					&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(t.memory.min * 65536)},
				},
			}},
		})
	}

	if len(t.elements) > 0 {
		elts := make([]ast.Expr, len(t.elements))
		for i, elem := range t.elements {
			inner := make([]ast.Expr, len(elem.init))
			for j, idx := range elem.init {
				inner[j] = &ast.SelectorExpr{X: newID("m"), Sel: t.functions[idx].decl.Name}
			}
			elts[i] = &ast.CompositeLit{
				Type: &ast.ArrayType{Elt: newID("any")},
				Elts: inner,
			}
		}
		body.List = append(body.List, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.SelectorExpr{X: newID("m"), Sel: newID("elements")}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.CompositeLit{
				Type: &ast.ArrayType{Elt: &ast.ArrayType{Elt: newID("any")}},
				Elts: elts,
			}},
		})

		for i, elem := range t.elements {
			if elem.passive {
				continue
			}
			body.List = append(body.List, &ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: newID("copy"),
					Args: []ast.Expr{
						&ast.SliceExpr{
							X:   &ast.SelectorExpr{X: newID("m"), Sel: t.table.id},
							Low: &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(int(elem.offset))},
						},
						&ast.IndexExpr{
							X:     &ast.SelectorExpr{X: newID("m"), Sel: newID("elements")},
							Index: &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(i)},
						},
					},
				},
			})
		}
	}

	for _, g := range t.globals {
		body.List = append(body.List, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.SelectorExpr{
				X:   newID("m"),
				Sel: g.id,
			}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{g.init},
		})
	}

	for i, seg := range t.data {
		if seg.passive {
			continue
		}
		body.List = append(body.List, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: newID("copy"),
				Args: []ast.Expr{
					&ast.SliceExpr{
						X:   &ast.SelectorExpr{X: newID("m"), Sel: t.memory.id},
						Low: &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(int(seg.offset))},
					},
					newID(fmt.Sprintf("data%d", i)),
				},
			},
		})
	}

	if t.start != 0 {
		body.List = append(body.List, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   newID("m"),
					Sel: t.functions[^t.start].decl.Name,
				},
			},
		})
	}

	body.List = append(body.List, &ast.ReturnStmt{Results: []ast.Expr{newID("m")}})

	return &ast.FuncDecl{
		Name: newID("New"),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: params},
			Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: newID("Module")}}}},
		},
		Body: body,
	}
}
