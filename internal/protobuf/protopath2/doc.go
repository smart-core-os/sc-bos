// Package protopath2 provides path parsing and traversal.
// The package is copied from https://go-review.googlesource.com/c/protobuf/+/582735 which is has not yet landed in the main protobuf module.
// See https://github.com/golang/protobuf/issues/1612 for discussion.
//
// Changes made to the original code:
// - Changed module path to github.com/smart-core-os/sc-bos/internal/protobuf/protopath2
// - Added support for parsing paths that use JSON names for field access (parse.go, parse_test.go, and testmessage.proto).
package protopath2
