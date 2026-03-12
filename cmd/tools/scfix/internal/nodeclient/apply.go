package nodeclient

import (
	"go/ast"
	"go/token"
)

// applyReplacement modifies the AST to replace n.Client(&x) with x := pkg.NewClient(n.ClientConn())
func applyReplacement(repl replacement) {
	assignToken := determineAssignmentToken(repl)
	newStmt := repl.newAssignment(assignToken)
	if newStmt == nil {
		return
	}

	varDeclIndex := findVarDeclToRemove(repl, assignToken)
	repl.blockStmt.List = buildNewStatementList(repl, newStmt, varDeclIndex)
}

// determineAssignmentToken decides whether to use := or = for the assignment
func determineAssignmentToken(repl replacement) token.Token {
	ident, ok := repl.clientRef.(*ast.Ident)
	if !ok {
		return token.ASSIGN // Struct fields and selectors always use =
	}

	if isFileLevelVar(repl.blockStmt, ident.Name) {
		return token.ASSIGN // Global variables use =
	}

	// Check for single-variable var declaration immediately before
	if repl.assignIndex > 0 {
		if isSingleVarDecl(repl.blockStmt.List[repl.assignIndex-1], ident.Name) {
			return token.DEFINE // Will remove var and use :=
		}
	}

	// For multi-var declarations or existing variables, use =
	if hasVarDeclInBlock(repl.blockStmt, ident.Name, repl.assignIndex) {
		return token.ASSIGN
	}

	// For new variables with :=, keep :=
	if repl.assignStmt != nil && repl.assignStmt.Tok == token.DEFINE {
		return token.DEFINE
	}

	return token.ASSIGN
}

// findVarDeclToRemove returns the index of a var declaration to remove, or -1
func findVarDeclToRemove(repl replacement, assignToken token.Token) int {
	if assignToken != token.DEFINE || repl.assignIndex == 0 {
		return -1
	}

	ident, ok := repl.clientRef.(*ast.Ident)
	if !ok {
		return -1
	}

	if isSingleVarDecl(repl.blockStmt.List[repl.assignIndex-1], ident.Name) {
		return repl.assignIndex - 1
	}

	return -1
}

// buildNewStatementList creates the new statement list with replacements applied
func buildNewStatementList(repl replacement, newStmt *ast.AssignStmt, varDeclIndex int) []ast.Stmt {
	newList := make([]ast.Stmt, 0, len(repl.blockStmt.List))

	for i, stmt := range repl.blockStmt.List {
		switch {
		case varDeclIndex >= 0 && i == varDeclIndex:
			// Skip var declaration - we're replacing it with :=
			continue
		case i == repl.assignIndex:
			// Replace the n.Client call with new assignment
			newList = append(newList, newStmt)
		case repl.hasError && i == repl.errorIndex:
			// Skip the error check statement
			continue
		default:
			newList = append(newList, stmt)
		}
	}

	return newList
}

// isSingleVarDecl checks if a statement is a single-variable var declaration for varName
func isSingleVarDecl(stmt ast.Stmt, varName string) bool {
	declStmt, ok := stmt.(*ast.DeclStmt)
	if !ok {
		return false
	}

	genDecl, ok := declStmt.Decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return false
	}

	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok || len(valueSpec.Names) != 1 {
			continue
		}

		if valueSpec.Names[0].Name == varName {
			return true
		}
	}

	return false
}

// hasVarDeclInBlock checks if a variable is declared anywhere in the block before a given index
func hasVarDeclInBlock(block *ast.BlockStmt, varName string, beforeIndex int) bool {
	for i := 0; i < beforeIndex; i++ {
		declStmt, ok := block.List[i].(*ast.DeclStmt)
		if !ok {
			continue
		}

		genDecl, ok := declStmt.Decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range valueSpec.Names {
				if name.Name == varName {
					return true
				}
			}
		}
	}

	return false
}

// isFileLevelVar checks if a variable is declared at file level (not in the current block)
func isFileLevelVar(block *ast.BlockStmt, varName string) bool {
	for _, stmt := range block.List {
		declStmt, ok := stmt.(*ast.DeclStmt)
		if !ok {
			continue
		}

		genDecl, ok := declStmt.Decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range valueSpec.Names {
				if name.Name == varName {
					return false // Found in block, not file-level
				}
			}
		}
	}

	return true // Not found in block, must be file-level
}
