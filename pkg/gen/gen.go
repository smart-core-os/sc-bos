package gen

// PREREQUISITE: protoc-gen-router is on PATH, i.e. `go install github.com/smart-core-os/sc-golang/cmd/protoc-gen-router@latest`
// PREREQUISITE: protoc-gen-wrapper is on PATH, i.e. `go install github.com/smart-core-os/sc-golang/cmd/protoc-gen-wrapper@latest`
//go:generate go run ../../cmd/tools/genproto -only goproto
