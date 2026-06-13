package passes

import (
	"go/ast"
	"go/token"
)

// GroupDecls coalesces single-variable declarations on function entry into
// single-declaration-per-type lines.
//
//	var t0 int32
//	var t3 int32     =>    var t0, t3 int32
//	var t1 int64           var t1 int64
func GroupDecls(fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}
	// Aggregate all leading `var x T` decls.
	var specs []*ast.ValueSpec
	for _, s := range fn.Body.List {
		if ds, ok := s.(*ast.DeclStmt); !ok {
			break
		} else if gd, ok := ds.Decl.(*ast.GenDecl); !ok || gd.Tok != token.VAR || len(gd.Specs) != 1 {
			break
		} else if vs, ok := gd.Specs[0].(*ast.ValueSpec); !ok || len(vs.Values) != 0 {
			break
		} else if _, ok := vs.Type.(*ast.Ident); !ok {
			break
		} else {
			specs = append(specs, vs)
		}
	}
	if len(specs) < 2 {
		return
	}
	// Merge names into one declaration per type, in first-seen order.
	byType := map[string]*ast.ValueSpec{}
	var grouped []ast.Stmt
	for _, vs := range specs {
		t := vs.Type.(*ast.Ident).Name
		if spec := byType[t]; spec != nil {
			spec.Names = append(spec.Names, vs.Names...)
			continue
		}
		spec := &ast.ValueSpec{Names: vs.Names, Type: vs.Type}
		byType[t] = spec
		grouped = append(grouped, &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{spec}}})
	}
	fn.Body.List = append(grouped, fn.Body.List[len(specs):]...)
}
