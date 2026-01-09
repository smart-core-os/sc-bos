package nodeclient

import (
	"go/ast"
	"go/token"
)

// findReplacements searches a file for n.Client(&x) calls that need to be replaced
func findReplacements(file *ast.File) []replacement {
	var replacements []replacement

	ast.Inspect(file, func(n ast.Node) bool {
		repl := tryExtractReplacement(file, n)
		if repl != nil {
			replacements = append(replacements, *repl)
		}
		return true
	})

	return replacements
}

// tryExtractReplacement attempts to extract a replacement from an AST node
func tryExtractReplacement(file *ast.File, n ast.Node) *replacement {
	callExpr, parentStmt, assignStmt := extractClientCall(n)
	if callExpr == nil {
		return nil
	}

	clientRef := extractClientRef(callExpr)
	if clientRef == nil {
		return nil
	}

	clientType := findClientType(file, clientRef)
	if clientType == nil {
		return nil
	}

	parentBlock, stmtIndex := findParentBlock(file, parentStmt)
	if parentBlock == nil {
		return nil
	}

	selExpr := callExpr.Fun.(*ast.SelectorExpr)
	hasError, errorIndex := checkForErrorHandling(parentBlock, stmtIndex, assignStmt)

	return &replacement{
		blockStmt:   parentBlock,
		stmt:        parentStmt,
		assignStmt:  assignStmt,
		assignIndex: stmtIndex,
		errorIndex:  errorIndex,
		hasError:    hasError,
		clientRef:   clientRef,
		clientType:  clientType,
		receiver:    selExpr.X,
	}
}

// extractClientCall checks if a node is a Client() call and extracts relevant info
func extractClientCall(n ast.Node) (callExpr *ast.CallExpr, parentStmt ast.Stmt, assignStmt *ast.AssignStmt) {
	// Check for assignment: err := n.Client(&client)
	if stmt, ok := n.(*ast.AssignStmt); ok {
		if len(stmt.Rhs) == 1 {
			if call, ok := stmt.Rhs[0].(*ast.CallExpr); ok && isClientCall(call) {
				return call, stmt, stmt
			}
		}
	}

	// Check for expression statement: n.Client(&client)
	if stmt, ok := n.(*ast.ExprStmt); ok {
		if call, ok := stmt.X.(*ast.CallExpr); ok && isClientCall(call) {
			return call, stmt, nil
		}
	}

	return nil, nil, nil
}

// isClientCall checks if a call expression is a .Client() method call
func isClientCall(call *ast.CallExpr) bool {
	selExpr, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selExpr.Sel.Name == "Client"
}

// extractClientRef gets the client reference from n.Client(&ref)
func extractClientRef(callExpr *ast.CallExpr) ast.Expr {
	if len(callExpr.Args) != 1 {
		return nil
	}

	unaryExpr, ok := callExpr.Args[0].(*ast.UnaryExpr)
	if !ok || unaryExpr.Op != token.AND {
		return nil
	}

	return unaryExpr.X
}

// checkForErrorHandling determines if there's an error check after the assignment
func checkForErrorHandling(block *ast.BlockStmt, stmtIndex int, assignStmt *ast.AssignStmt) (bool, int) {
	errorIndex := stmtIndex + 1
	if assignStmt == nil || errorIndex >= len(block.List) || len(assignStmt.Lhs) == 0 {
		return false, errorIndex
	}

	ifStmt, ok := block.List[errorIndex].(*ast.IfStmt)
	if !ok {
		return false, errorIndex
	}

	return isErrorCheck(ifStmt, assignStmt.Lhs[0]), errorIndex
}

// isErrorCheck verifies if an if statement checks for err != nil
func isErrorCheck(ifStmt *ast.IfStmt, errVar ast.Expr) bool {
	binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok || binExpr.Op != token.NEQ {
		return false
	}

	ident, ok := binExpr.X.(*ast.Ident)
	if !ok {
		return false
	}

	errIdent, ok := errVar.(*ast.Ident)
	if !ok || ident.Name != errIdent.Name {
		return false
	}

	nilIdent, ok := binExpr.Y.(*ast.Ident)
	return ok && nilIdent.Name == "nil"
}

// findParentBlock finds the block statement containing the given statement
func findParentBlock(file *ast.File, target ast.Stmt) (*ast.BlockStmt, int) {
	var parentBlock *ast.BlockStmt
	var stmtIndex int

	ast.Inspect(file, func(n ast.Node) bool {
		if parentBlock != nil {
			return false
		}

		blockStmt, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}

		for i, stmt := range blockStmt.List {
			if stmt == target {
				parentBlock = blockStmt
				stmtIndex = i
				return false
			}
		}

		return true
	})

	return parentBlock, stmtIndex
}
