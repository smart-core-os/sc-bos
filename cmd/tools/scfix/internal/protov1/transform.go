package protov1

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	fieldRe     = regexp.MustCompile(`(?m)^(\s*(?:(?:repeated|optional)\s+)?)([\w.]+)(\s+\w+\s*=\s*\d+;)`)
	goPackageRe = regexp.MustCompile(`(?m)^option\s+go_package\s*=\s*"[^"]*"\s*;`)

	// rpcDeclRe matches proto RPC declaration lines, capturing the input and output type references.
	// Handles both streaming and non-streaming variants.
	// Groups: (rpc prefix)(InputType)(between)(OutputType)(line suffix)
	rpcDeclRe = regexp.MustCompile(`(?m)^(\s*rpc\s+\w+\s*\((?:stream\s+)?)([\w.]+)(\s*\)\s*returns\s*\((?:stream\s+)?)([\w.]+)(\).*)`)
)

// buildImportedTypesForFile returns a mapping from type->package for all files in allFiles imported by content.
func buildImportedTypesForFile(content []byte, allFiles []protoFile) map[string]string {
	importedTypes := make(map[string]string)
	imports := extractImports(content)

	// Build lookup maps for finding a protoFile given its import path.
	// fileMapByPath: keyed by getOldImportPath() — works for proto-root files where the
	//   import path is relative to the proto root directory.
	// fileMapByAlt: keyed by altImportPaths — works for sc-api source files where the
	//   import path is relative to the sc-api protobuf include dir (e.g. "types/unit.proto").
	// fileMapByNewPath: keyed by getNewImportPath() — resolves imports that have already been
	//   updated by a prior fixer run (e.g. "smartcore/bos/types/v1/unit.proto" for a moved file).
	// No basename fallback: basename collisions (e.g. health.proto in both info/ and health/)
	// would cause incorrect type resolution.
	fileMapByPath := make(map[string]*protoFile)
	fileMapByAlt := make(map[string]*protoFile)
	fileMapByNewPath := make(map[string]*protoFile)
	for i := range allFiles {
		fileMapByPath[allFiles[i].getOldImportPath()] = &allFiles[i]
		fileMapByNewPath[allFiles[i].getNewImportPath()] = &allFiles[i]
		for _, alt := range allFiles[i].altImportPaths {
			fileMapByAlt[alt] = &allFiles[i]
		}
	}

	for _, importPath := range imports {
		// Try direct lookup first (by old import path or alt import path).
		file, ok := fileMapByPath[importPath]
		if !ok {
			file, ok = fileMapByAlt[importPath]
		}
		// If not found, the import may refer to a file using its new (canonical) path
		// (e.g. after a prior fixer run already updated some imports), so try the new path map.
		if !ok {
			file, ok = fileMapByNewPath[importPath]
		}
		if ok {
			// Add all types from this imported file - they all share the same package
			for _, typeName := range file.types {
				importedTypes[typeName] = file.newPackage
			}
		}
	}

	return importedTypes
}

// processProtoFile updates the content of a proto file with new package declaration and import paths.
func processProtoFile(file *protoFile, allFiles []protoFile) error {
	if file.oldPackage == "" {
		return fmt.Errorf("no package declaration found")
	}

	// Build a map of types that are imported in this file, then overwrite with local types.
	// Proto resolves unqualified names in the current package first, so local definitions must
	// take priority to prevent unqualified refs from gaining an incorrect package qualifier.
	// Qualified refs to smartcore.types.* / smartcore.info.* are handled by package-level
	// remapping in resolveTypeRef and do not use this map.
	importedTypes := buildImportedTypesForFile(file.oldContent, allFiles)
	for _, typeName := range file.types {
		importedTypes[typeName] = file.newPackage
	}

	content := string(file.oldContent)
	content = updatePackageDeclaration(content, file.oldPackage, file.newPackage)
	content = updateServiceDeclarations(content, file.serviceRenames)
	content = updateImportPaths(content, allFiles)
	content = updateTypeReferences(content, file.newPackage, importedTypes)
	if file.newGoPackage != "" {
		content = updateGoPackageOption(content, file.newGoPackage)
	}

	file.newContent = []byte(content)
	return nil
}

// updateGoPackageOption replaces the go_package option value with newGoPackage.
func updateGoPackageOption(content, newGoPackage string) string {
	return goPackageRe.ReplaceAllString(content, fmt.Sprintf(`option go_package = "%s";`, newGoPackage))
}

// updatePackageDeclaration updates the package declaration from oldPkg to newPkg.
func updatePackageDeclaration(content, oldPkg, newPkg string) string {
	if oldPkg == newPkg {
		return content
	}
	re := regexp.MustCompile(`(?m)^package\s+` + regexp.QuoteMeta(oldPkg) + `\s*;`)
	return re.ReplaceAllString(content, fmt.Sprintf("package %s;", newPkg))
}

// updateServiceDeclarations updates all service declarations in the file using the serviceRenames map.
func updateServiceDeclarations(content string, serviceRenames map[string]string) string {
	if len(serviceRenames) == 0 {
		return content
	}

	// Find all matches and their positions
	matches := serviceRe.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content
	}

	// Build the result by replacing matches in reverse order (to preserve indices)
	result := content
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		// match[0], match[1] are the full match start/end
		// match[2], match[3] are the service name (first capture group) start/end
		if len(match) < 4 {
			continue
		}

		oldService := content[match[2]:match[3]]
		if newService, shouldRename := serviceRenames[oldService]; shouldRename {
			// Replace the entire service declaration with the renamed version
			replacement := fmt.Sprintf("service %s {", newService)
			result = result[:match[0]] + replacement + result[match[1]:]
		}
	}

	return result
}

// updateImportPaths rewrites local proto imports to use versioned paths and sorts imports alphabetically.
func updateImportPaths(content string, allFiles []protoFile) string {
	// Build lookup maps: old import path -> new import path (only for moved files).
	//
	// pathMap: keyed by getOldImportPath() — matches proto-root files where import strings are
	//   paths relative to the proto root (e.g. "smartcore/bos/health/v1/health.proto").
	// altMap: keyed by altImportPaths — matches sc-api files where import strings are paths
	//   relative to the sc-api protobuf include dir (e.g. "types/unit.proto", "info/health.proto").
	//
	// No basename fallback: basename collisions (e.g. health.proto existing in both the info/
	// and health/ directories) would corrupt already-versioned imports that happen to share a filename.
	pathMap := make(map[string]string)
	altMap := make(map[string]string)
	for _, file := range allFiles {
		newImp := file.getNewImportPath()
		if file.oldPath != file.newPath {
			pathMap[file.getOldImportPath()] = newImp
		}
		// Alt import paths (e.g. sc-api-relative or legacy paths) should always be remapped
		// to the canonical new import path, regardless of whether the file is moving.
		// This handles files already at their final location that still have legacy alt paths
		// (e.g. "types/unit.proto" → "smartcore/bos/types/v1/unit.proto").
		for _, alt := range file.altImportPaths {
			altMap[alt] = newImp
		}
	}

	lines := strings.Split(content, "\n")
	var result []string
	var importLines []string
	inImportSection := false

	for _, line := range lines {
		matches := importRe.FindStringSubmatch(line)
		if matches != nil {
			inImportSection = true
			importPath := matches[2]

			var newPath string
			if p, exists := pathMap[importPath]; exists {
				newPath = p
			} else if p, exists := altMap[importPath]; exists {
				newPath = p
			}
			if newPath != "" && importPath != newPath {
				line = matches[1] + `"` + newPath + `";`
			}

			importLines = append(importLines, line)
			continue
		}
		if inImportSection {
			sort.Strings(importLines)
			result = append(result, importLines...)
			importLines = nil
			inImportSection = false
		}
		result = append(result, line)

	}

	if inImportSection {
		sort.Strings(importLines)
		result = append(result, importLines...)
	}

	return strings.Join(result, "\n")
}

// updateTypeReferences updates type references in field declarations and RPC signatures to use
// fully qualified package names when the type is from a different package.
func updateTypeReferences(content string, currentPackage string, typeToPackage map[string]string) string {
	content = updateFieldTypeReferences(content, currentPackage, typeToPackage)
	content = updateRPCTypeReferences(content, currentPackage, typeToPackage)
	return content
}

// resolveTypeRef resolves a proto type reference (possibly qualified) to its correct form.
// If the type is in currentPackage, the package qualifier is stripped (returns just typeName).
// If the type is in a different known package, the correct package qualifier is added.
// Returns the resolved reference, or "" if the type is unknown and should be left unchanged.
func resolveTypeRef(typeName string, currentPackage string, typeToPackage map[string]string) string {
	builtInTypes := map[string]bool{
		"double": true, "float": true, "int32": true, "int64": true,
		"uint32": true, "uint64": true, "sint32": true, "sint64": true,
		"fixed32": true, "fixed64": true, "sfixed32": true, "sfixed64": true,
		"bool": true, "string": true, "bytes": true,
	}

	packagePart, unqualifiedType, isQualified := cutLast(typeName, ".")

	if isQualified {
		// For types/info packages that move as a unit, remap by package alone — no type lookup.
		// This avoids incorrectly resolving local messages that share a name with an imported type
		// (e.g. temperature.proto's local Temperature vs smartcore.types.Temperature).
		var newPkg string
		if packagePart == "smartcore.info" {
			newPkg = "smartcore.bos.info.v1"
		} else if packagePart == "smartcore.types" {
			newPkg = "smartcore.bos.types.v1"
		} else if strings.HasPrefix(packagePart, "smartcore.types.") {
			sub := strings.TrimPrefix(packagePart, "smartcore.types.")
			newPkg = "smartcore.bos.types." + sub + ".v1"
		}
		if newPkg != "" {
			if newPkg == currentPackage {
				return unqualifiedType
			}
			return newPkg + "." + unqualifiedType
		}

		// For traits and unversioned BOS packages: use type-name lookup.
		needsUpdate := (strings.HasPrefix(packagePart, "smartcore.bos") && !strings.Contains(packagePart, ".v")) ||
			packagePart == "smartcore.traits"
		if needsUpdate {
			if typePackage, exists := typeToPackage[unqualifiedType]; exists {
				if typePackage == currentPackage {
					return unqualifiedType
				}
				return typePackage + "." + unqualifiedType
			}
		}
		return "" // leave qualified ref unchanged
	}

	if builtInTypes[typeName] {
		return "" // leave built-ins unchanged
	}

	typePackage, exists := typeToPackage[typeName]
	if !exists || typePackage == currentPackage {
		return "" // unknown or same-package — no change
	}
	return typePackage + "." + typeName
}

// updateFieldTypeReferences updates type references in message field declarations.
func updateFieldTypeReferences(content string, currentPackage string, typeToPackage map[string]string) string {
	return fieldRe.ReplaceAllStringFunc(content, func(match string) string {
		submatches := fieldRe.FindStringSubmatch(match)
		if len(submatches) < 4 {
			return match
		}
		prefix := submatches[1]
		typeName := submatches[2]
		suffix := submatches[3]

		if resolved := resolveTypeRef(typeName, currentPackage, typeToPackage); resolved != "" {
			return prefix + resolved + suffix
		}
		return match
	})
}

// updateRPCTypeReferences updates type references in RPC declarations (input and output types).
// These are not matched by the field regex since RPC lines use parentheses, not `= N;` suffixes.
func updateRPCTypeReferences(content string, currentPackage string, typeToPackage map[string]string) string {
	return rpcDeclRe.ReplaceAllStringFunc(content, func(match string) string {
		submatches := rpcDeclRe.FindStringSubmatch(match)
		if len(submatches) < 6 {
			return match
		}
		pre := submatches[1]
		inputType := submatches[2]
		between := submatches[3]
		outputType := submatches[4]
		suf := submatches[5]

		if r := resolveTypeRef(inputType, currentPackage, typeToPackage); r != "" {
			inputType = r
		}
		if r := resolveTypeRef(outputType, currentPackage, typeToPackage); r != "" {
			outputType = r
		}
		return pre + inputType + between + outputType + suf
	})
}

// cutLast splits s around the last instance of sep.
// Like strings.Cut but operates on the last occurrence instead of the first.
func cutLast(s, sep string) (before, after string, found bool) {
	if i := strings.LastIndex(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
