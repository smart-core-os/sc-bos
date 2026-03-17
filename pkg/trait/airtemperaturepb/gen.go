package airtemperaturepb

// PREREQUISITE: protomod is on PATH, i.e. `go install github.com/smart-core-os/protomod`
// PREREQUISITE: protoc-gen-router is on PATH, i.e. `go install github.com/smart-core-os/sc-bos/cmd/tools/protoc-gen-router`
// PREREQUISITE: protoc-gen-wrapper is on PATH, i.e. `go install github.com/smart-core-os/sc-bos/cmd/tools/protoc-gen-wrapper`
//go:generate protomod protoc -- -I ../../.. --router_opt=outputPkg=github.com/smart-core-os/sc-bos/pkg/trait/airtemperaturepb --router_out=../../.. --wrapper_opt=outputPkg=github.com/smart-core-os/sc-bos/pkg/trait/airtemperaturepb --wrapper_out=../../.. github.com/smart-core-os/sc-bos/sc-api/protobuf/traits/air_temperature.proto
