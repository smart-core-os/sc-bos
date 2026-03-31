package testproto

//go:generate protoc -I../.. --go_out=paths=source_relative:../.. --go-grpc_out=paths=source_relative:../.. internal/testproto/test.proto
