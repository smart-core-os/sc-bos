package nodeclient

import (
	"go/ast"
	"go/token"
)

// extractClientTypeFromExpr extracts client type info from a type expression
func extractClientTypeFromExpr(typeExpr ast.Expr) *clientTypeInfo {
	selExpr, ok := typeExpr.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	pkgIdent, ok := selExpr.X.(*ast.Ident)
	if !ok {
		return nil
	}

	typeName := selExpr.Sel.Name
	constructor := "New" + typeName

	return &clientTypeInfo{
		pkgName:     pkgIdent.Name,
		typeName:    typeName,
		constructor: constructor,
	}
}

// findClientType finds the type information for a client reference
func findClientType(file *ast.File, clientRef ast.Expr) *clientTypeInfo {
	switch ref := clientRef.(type) {
	case *ast.Ident:
		return findIdentType(file, ref.Name)
	case *ast.SelectorExpr:
		return findSelectorType(file, ref)
	default:
		return nil
	}
}

// findIdentType finds the type of a local or global variable by name
func findIdentType(file *ast.File, varName string) *clientTypeInfo {
	var result *clientTypeInfo
	ast.Inspect(file, func(n ast.Node) bool {
		if result != nil {
			return false
		}

		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			return true
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range valueSpec.Names {
				if name.Name == varName && valueSpec.Type != nil {
					result = extractClientTypeFromExpr(valueSpec.Type)
					return false
				}
			}
		}

		return true
	})
	return result
}

// findSelectorType finds the type of a struct field (e.g., s.client)
func findSelectorType(file *ast.File, sel *ast.SelectorExpr) *clientTypeInfo {
	baseIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil
	}

	structTypeName := findStructTypeName(file, baseIdent.Name)
	if structTypeName == "" {
		return nil
	}

	return findFieldType(file, structTypeName, sel.Sel.Name)
}

// findStructTypeName finds the type name of a variable
func findStructTypeName(file *ast.File, varName string) string {
	var structTypeName string
	ast.Inspect(file, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			return true
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for i, name := range valueSpec.Names {
				if name.Name == varName {
					if valueSpec.Type != nil {
						if ident, ok := valueSpec.Type.(*ast.Ident); ok {
							structTypeName = ident.Name
						}
					} else if len(valueSpec.Values) > i {
						if compLit, ok := valueSpec.Values[i].(*ast.CompositeLit); ok {
							if ident, ok := compLit.Type.(*ast.Ident); ok {
								structTypeName = ident.Name
							}
						}
					}
					return false
				}
			}
		}

		return true
	})

	return structTypeName
}

// findFieldType finds the type of a field in a struct
func findFieldType(file *ast.File, structTypeName, fieldName string) *clientTypeInfo {
	var result *clientTypeInfo
	ast.Inspect(file, func(n ast.Node) bool {
		if result != nil {
			return false
		}

		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != structTypeName {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		for _, field := range structType.Fields.List {
			for _, fieldIdent := range field.Names {
				if fieldIdent.Name == fieldName {
					result = extractClientTypeFromExpr(field.Type)
					return false
				}
			}
		}

		return true
	})

	return result
}
