package protov1

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	fieldRe = regexp.MustCompile(`(?m)^(\s*(?:(?:repeated|optional)\s+)?)([\w.]+)(\s+\w+\s*=\s*\d+;)`)
)

// buildImportedTypesForFile returns a mapping from type->package for all files in allFiles imported by content.
func buildImportedTypesForFile(content []byte, allFiles []protoFile) map[string]string {
	importedTypes := make(map[string]string)
	imports := extractImports(content)

	// Build a quick lookup map: old import path -> protoFile
	fileMap := make(map[string]*protoFile)
	for i := range allFiles {
		importPath := allFiles[i].getOldImportPath()
		fileMap[importPath] = &allFiles[i]
	}

	for _, importPath := range imports {
		if file, exists := fileMap[importPath]; exists {
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

	// Build a map of types that are imported in this file
	importedTypes := buildImportedTypesForFile(file.oldContent, allFiles)

	content := string(file.oldContent)
	content = updatePackageDeclaration(content, file.oldPackage, file.newPackage)
	content = updateServiceDeclarations(content, file.serviceRenames)
	content = updateImportPaths(content, allFiles)
	content = updateTypeReferences(content, file.newPackage, importedTypes)

	file.newContent = []byte(content)
	return nil
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
	// Build a quick lookup map: filename -> new import path (only for moved files)
	fileMap := make(map[string]string)
	for _, file := range allFiles {
		if file.oldPath != file.newPath {
			// If the file hasn't moved, the import won't have changed,
			// if the file isn't in fileMap the import line won't be changed.
			filename := filepath.Base(file.oldPath)
			fileMap[filename] = file.getNewImportPath()
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
			filename := filepath.Base(importPath)

			if newPath, exists := fileMap[filename]; exists {
				if !strings.Contains(importPath, "/") {
					line = matches[1] + `"` + newPath + `";`
				}
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

// updateTypeReferences updates type references in field declarations to use fully qualified package names
// when the type is from a different package.
func updateTypeReferences(content string, currentPackage string, typeToPackage map[string]string) string {
	builtInTypes := map[string]bool{
		"double": true, "float": true, "int32": true, "int64": true,
		"uint32": true, "uint64": true, "sint32": true, "sint64": true,
		"fixed32": true, "fixed64": true, "sfixed32": true, "sfixed64": true,
		"bool": true, "string": true, "bytes": true,
	}

	return fieldRe.ReplaceAllStringFunc(content, func(match string) string {
		submatches := fieldRe.FindStringSubmatch(match)
		if len(submatches) < 4 {
			return match
		}

		prefix := submatches[1]
		typeName := submatches[2]
		suffix := submatches[3]
		packagePart, unqualifiedType, isQualified := cutLast(typeName, ".")

		if isQualified {
			if strings.HasPrefix(packagePart, "smartcore.bos") && !strings.Contains(packagePart, ".v") {
				if typePackage, exists := typeToPackage[unqualifiedType]; exists {
					return prefix + typePackage + "." + unqualifiedType + suffix
				}
			}
			return match
		}

		if builtInTypes[typeName] {
			return match
		}

		typePackage, exists := typeToPackage[typeName]
		if !exists || typePackage == currentPackage {
			return match
		}

		return prefix + typePackage + "." + typeName + suffix
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
