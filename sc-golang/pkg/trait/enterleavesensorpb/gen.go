package enterleavesensorpb

// PREREQUISITE: protomod is on PATH, i.e. `go install git.vanti.co.uk/vanti-incubator/protomod`
// PREREQUISITE: protoc-gen-router is on PATH, i.e. `go install github.com/smart-core-os/sc-golang/cmd/protoc-gen-router`
// PREREQUISITE: protoc-gen-wrapper is on PATH, i.e. `go install github.com/smart-core-os/sc-golang/cmd/protoc-gen-wrapper`
//go:generate protomod protoc -- -I ../../.. --router_out=../../.. --wrapper_out=../../.. github.com/smart-core-os/sc-api/protobuf/traits/enter_leave_sensor.proto
