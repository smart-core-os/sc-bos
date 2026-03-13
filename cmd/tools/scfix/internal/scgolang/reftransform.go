package scgolang

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

// transformImports rewrites sc-golang import paths in a Go source file to their
// sc-bos equivalents. Removed packages are silently left unchanged.
// Returns the (possibly updated) content, whether any changes were made, and any error.
func transformImports(content string) (string, bool, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return content, false, err
	}

	changed := false
	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if !strings.HasPrefix(path, oldModule+"/") {
			continue
		}
		newPath, ok := scgolangImportToScBos(path)
		if !ok || newPath == path {
			// Either a removed package (ok=false, newPath="") or not an sc-golang import.
			// Leave unchanged in both cases.
			continue
		}
		imp.Path.Value = `"` + newPath + `"`
		changed = true
	}

	if !changed {
		return content, false, nil
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return content, false, err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return content, false, err
	}

	return string(formatted), true, nil
}
