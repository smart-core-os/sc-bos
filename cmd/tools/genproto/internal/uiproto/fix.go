package uiproto

import (
	"regexp"
)

var (
	// jsImportPattern matches CommonJS require statements for traits, types, and info imports.
	// Example: require('./traits/foo_pb.js') or require('../../../traits/foo_pb.js')
	jsImportPattern = regexp.MustCompile(`require\('(?:\.\.?/)+((?:traits|types|info)\/.+_pb\.js)'\)`)

	// dtsImportPattern matches TypeScript import statements for traits, types, and info.
	// Example: from './traits/foo_pb' or from '../../../traits/foo_pb'
	dtsImportPattern = regexp.MustCompile(`from '(?:\.\.?/)+((?:traits|types|info)\/.+_pb)'`)
)

// fixJSImports replaces relative imports with package imports in JavaScript files.
// Transforms: require('./traits/foo_pb.js') -> require('@smart-core-os/sc-api-grpc-web/traits/foo_pb.js')
func fixJSImports(content []byte) []byte {
	return jsImportPattern.ReplaceAll(content, []byte(`require('@smart-core-os/sc-api-grpc-web/$1')`))
}

// fixDTSImports replaces relative imports with package imports in TypeScript definition files.
// Transforms: from './traits/foo_pb' -> from '@smart-core-os/sc-api-grpc-web/traits/foo_pb'
func fixDTSImports(content []byte) []byte {
	return dtsImportPattern.ReplaceAll(content, []byte(`from '@smart-core-os/sc-api-grpc-web/$1'`))
}
