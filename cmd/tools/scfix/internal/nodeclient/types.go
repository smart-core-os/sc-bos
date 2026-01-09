package nodeclient

import (
	"go/ast"
	"go/token"
)

// replacement represents a single n.Client(&x) call that needs to be replaced.
//
// Example input code being replaced:
//
//	var client traits.OnOffApiClient  // may be at assignIndex-1
//	err := n.Client(&client)          // stmt at assignIndex
//	if err != nil {                   // if hasError, at errorIndex
//	    return err
//	}
type replacement struct {
	blockStmt   *ast.BlockStmt  // The parent block containing all statements
	stmt        ast.Stmt        // The n.Client(...) statement: "err := n.Client(&client)" or "n.Client(&client)"
	assignStmt  *ast.AssignStmt // The assignment if stmt is AssignStmt: "err := n.Client(&client)", nil otherwise
	assignIndex int             // Index of stmt in blockStmt.List
	errorIndex  int             // Index of error check block if present: "if err != nil { ... }"
	hasError    bool            // Whether an error check block exists at errorIndex
	clientRef   ast.Expr        // The client reference: "client" or "s.client"
	clientType  *clientTypeInfo // Type info for generating constructor: traits.OnOffApiClient -> NewOnOffApiClient
	receiver    ast.Expr        // The receiver of Client call: "n" in "n.Client(&client)"
}

// clientTypeInfo holds information about a client type for code generation
type clientTypeInfo struct {
	pkgName     string // e.g., "traits"
	typeName    string // e.g., "OnOffApiClient"
	constructor string // e.g., "NewOnOffApiClient"
}

// newClientCall creates the AST for traits.NewOnOffApiClient(n.ClientConn())
func (c *clientTypeInfo) newClientCall(receiver ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent(c.pkgName),
			Sel: ast.NewIdent(c.constructor),
		},
		Args: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   receiver,
					Sel: ast.NewIdent("ClientConn"),
				},
				Args: []ast.Expr{},
			},
		},
	}
}

// newAssignment creates an assignment statement for the client
func (r *replacement) newAssignment(assignToken token.Token) *ast.AssignStmt {
	switch ref := r.clientRef.(type) {
	case *ast.Ident:
		return &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent(ref.Name)},
			Tok: assignToken,
			Rhs: []ast.Expr{r.clientType.newClientCall(r.receiver)},
		}
	case *ast.SelectorExpr:
		return &ast.AssignStmt{
			Lhs: []ast.Expr{ref},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{r.clientType.newClientCall(r.receiver)},
		}
	default:
		return nil
	}
}
