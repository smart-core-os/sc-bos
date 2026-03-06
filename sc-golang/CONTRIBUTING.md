# Contributing to this project

## Automated tasks

Routers and Wrappers are both generated from the underlying `.proto` files via protoc plugins. The plugins output files
like `foo_router.pb.go` or `bar_wrap.pb.go`.

The source for the plugins can be found in `cmd/protoc-gen-...` directories. Install the plugins into your PATH
via `go install ./cmd/protoc-gen-router`, using similar commands for each plugin.

The plugins are generally invokes via `go generate ./...`, a `gen.go` file exists in each of the `pkg/trait/{trait}`
packages. See there for prerequisites of each trait tools. 
