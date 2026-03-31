package goprotoimports

import (
	"go/ast"
	"regexp"
)

// updateCommentRefs rewrites pkgAlias.Symbol references in comments to use the new package aliases.
// Symbols that appeared only in comments (not in code) are resolved on-demand via resolve.
// Returns the number of comments modified.
func updateCommentRefs(
	node *ast.File,
	pkgAlias string,
	resolved map[string]symDest,
	pkgToAlias map[string]string,
	resolve func(string) (symDest, bool),
) int {
	changes := 0
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(pkgAlias) + `\.([A-Z][a-zA-Z0-9_]*)`)
	for _, commentGroup := range node.Comments {
		for _, comment := range commentGroup.List {
			updated := pattern.ReplaceAllStringFunc(comment.Text, func(match string) string {
				symbol := pattern.FindStringSubmatch(match)[1]
				dest, ok := resolved[symbol]
				if !ok {
					// Symbol appeared in a comment but not in code — resolve directly.
					dest, ok = resolve(symbol)
					if !ok {
						return match // unknown, leave as-is
					}
				}
				if dest.self {
					return dest.symbol
				}
				alias, ok := pkgToAlias[dest.pkgName]
				if !ok {
					// Destination package wasn't referenced in code; use unaliased name.
					alias = dest.pkgName
				}
				return alias + "." + dest.symbol
			})
			if updated != comment.Text {
				comment.Text = updated
				changes++
			}
		}
	}
	return changes
}
