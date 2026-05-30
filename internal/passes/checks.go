package passes

import (
	"go/ast"
	"go/token"
	"strings"
)

// CheckMaterialized verifies that materialized constants
// (i.e. locals starting with a t) are never reassigned,
// only defined.
func CheckMaterialized(n ast.Node) {
	ast.Inspect(n, func(n ast.Node) bool {
		if assign, ok := n.(*ast.AssignStmt); ok && assign.Tok == token.ASSIGN {
			for _, lhs := range assign.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && strings.HasPrefix(id.Name, "t") {
					panic("assignment to materialized constant: " + id.Name)
				}
			}
		}
		return true
	})
}
