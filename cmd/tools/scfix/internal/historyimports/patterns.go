package historyimports

import "regexp"

// Pattern to match single-line import statements from proto files
// Matches: import {A, B, C} from '@smart-core-os/sc-bos-ui-gen/proto/foo_pb';
// Also matches with .js extension: import {A, B} from '.../foo_pb.js';
var singleLineImportPattern = regexp.MustCompile(`^(\s*import\s+\{)([^}]+)(}\s+from\s+['"]@smart-core-os/sc-bos-ui-gen/proto/)([a-z_]+?)(_(?:grpc_web_)?pb)((?:\.js)?['"];?.*)$`)

// Pattern to match the start of a multi-line import
var multiLineStartPattern = regexp.MustCompile(`^(\s*import\s+\{)\s*$`)

// Pattern to match the end of a multi-line import with the from clause
var multiLineEndPattern = regexp.MustCompile(`^(\s*)(}\s+from\s+['"]@smart-core-os/sc-bos-ui-gen/proto/)([a-z_]+?)(_(?:grpc_web_)?pb)((?:\.js)?['"];?.*)$`)

// Pattern to match JSDoc with inline import() from proto files
// Matches: @param {import('...')}, @type {import('...')}, etc.
var jsdocImportPattern = regexp.MustCompile(`import\(['"]@smart-core-os/sc-bos-ui-gen/proto/([a-z_]+?)(_(?:grpc_web_)?pb)(?:\.js)?['"]\)`)

// Pattern to match proto file paths in JSDoc import statements
// Matches: proto/xxx_pb or proto/xxx_grpc_web_pb
var jsdocProtoPathPattern = regexp.MustCompile(`proto/([a-z_]+?)(_(?:grpc_web_)?pb)`)

// findMultiLineImportEnd finds the closing brace and from clause of a multi-line import.
func findMultiLineImportEnd(lines []string, startIdx int) int {
	for i := startIdx + 1; i < len(lines) && i < startIdx+50; i++ {
		if multiLineEndPattern.MatchString(lines[i]) {
			return i
		}
	}
	return -1
}
