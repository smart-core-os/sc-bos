package scgolang

import "strings"

const (
	oldModule = "github.com/smart-core-os/sc-golang"
	newModule = "github.com/smart-core-os/sc-bos"
)

// removedPrefixes are package path suffixes (relative to oldModule) that were
// deleted during the merge, not migrated. The fixer leaves these imports unchanged.
var removedPrefixes = []string{
	"pkg/server",
	"pkg/middleware",
	"pkg/client",
}

// scgolangImportToScBos converts an old sc-golang import path to its new sc-bos location.
// Returns ("", false) for removed packages (leave the import unchanged; user will see a compile error).
// Returns (importPath, false) for paths that are not sc-golang imports.
// Returns (newPath, true) when a rewrite is needed.
//
// Rules applied in order:
//  1. Must start with github.com/smart-core-os/sc-golang/ — if not, return unchanged
//  2. Check removed packages — return ("", false) to skip silently
//  3. internal/minibus → pkg/minibus
//  4. pkg/time or pkg/time/... → pkg/util/time or pkg/util/time/...
//  5. pkg/masks → pkg/util/masks
//  6. pkg/cmp → pkg/util/cmp
//  7. pkg/trait (exact) → pkg/trait
//  8. pkg/trait/<sub>/... where <sub> is in conflictingTraitPackages → pkg/trait/<sub>/...
//  9. pkg/trait/<sub>/... (all others) → pkg/proto/<sub>/...
//  10. Everything else → swap module prefix only
func scgolangImportToScBos(importPath string) (string, bool) {
	prefix := oldModule + "/"
	if !strings.HasPrefix(importPath, prefix) {
		return importPath, false
	}

	suffix := strings.TrimPrefix(importPath, prefix)

	// Check removed packages.
	for _, removed := range removedPrefixes {
		if suffix == removed || strings.HasPrefix(suffix, removed+"/") {
			return "", false
		}
	}

	// internal/minibus -> pkg/minibus
	if suffix == "internal/minibus" {
		return newModule + "/pkg/minibus", true
	}

	// pkg/time/... -> pkg/util/time/...
	if suffix == "pkg/time" || strings.HasPrefix(suffix, "pkg/time/") {
		rest := strings.TrimPrefix(suffix, "pkg/time")
		return newModule + "/pkg/util/time" + rest, true
	}

	// pkg/masks -> pkg/util/masks
	if suffix == "pkg/masks" {
		return newModule + "/pkg/util/masks", true
	}

	// pkg/cmp -> pkg/util/cmp
	if suffix == "pkg/cmp" {
		return newModule + "/pkg/util/cmp", true
	}

	// pkg/trait (exact) -> pkg/trait
	if suffix == "pkg/trait" {
		return newModule + "/pkg/trait", true
	}

	// pkg/trait/<sub>/...
	if strings.HasPrefix(suffix, "pkg/trait/") {
		sub := strings.TrimPrefix(suffix, "pkg/trait/")
		return newModule + "/pkg/proto/" + sub, true
	}

	// Everything else: just swap the module prefix.
	return newModule + "/" + suffix, true
}
