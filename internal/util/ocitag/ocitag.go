// Package ocitag validates OCI image tags. The software-update version doubles as the image tag the
// supervisor loads, so the supervisor and cloudsim share this one rule to reject a malformed version.
package ocitag

import "regexp"

// tagPattern matches a valid OCI image tag, per the reference grammar.
var tagPattern = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}$`)

// Valid reports whether s is a legal OCI image tag.
func Valid(s string) bool {
	return tagPattern.MatchString(s)
}
