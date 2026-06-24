package logdownload

//go:generate protoc -I ../.. -I ../../proto --go_out=paths=source_relative:../.. internal/logdownload/download.proto
