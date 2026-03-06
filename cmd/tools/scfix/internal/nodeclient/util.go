package nodeclient

import "go/ast"

// exprToString converts an expression to a string for logging
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	default:
		return "<expr>"
	}
}
