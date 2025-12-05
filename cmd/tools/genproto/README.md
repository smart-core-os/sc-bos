# genproto

Generates Go and JavaScript code from `.proto` files.

## Output

Proto files will produce different output files depending on their structure and contents.

For go files:

- All proto files will produce `.pb.go` and _grpc.pb.go files using `protoc-gen-go` and `protoc-gen-go-grpc`.
- Services where ALL request messages have a `string name` property will generate `_router.pb.go` using `protoc-gen-router`.
- Any proto file with a `service` definition will generate a `_wrapper.pb.go` using `protoc-gen-wrapper`.
- Generated files will be placed into the `pkg/gen` directory.

For js/ts files:

- All proto files will produce `_pb.js` and `_pb.d.ts` files using `protoc-gen-js`.
- All proto files will produce `_grpc_web_pb.js` and `_grpc_web_pb.d.ts` files using `protoc-gen-grpc-web`.
- Generated files will be placed into the `ui/ui-gen/proto` directory.

## Usage

From the repository root:

```bash
go run ./cmd/tools/genproto
```

Options:

```bash
go run ./cmd/tools/genproto -v          # verbose output
go run ./cmd/tools/genproto -dry-run    # preview without executing
go run ./cmd/tools/genproto -list       # list available generation steps
go run ./cmd/tools/genproto -only uiproto   # run only UI generation
go run ./cmd/tools/genproto -skip goproto   # skip Go generation
```

For JetBrains IDEs (GoLand, IntelliJ IDEA), use the "gen-proto" run configuration.

## Prerequisites

The following tools must be installed and available on your PATH:

```bash
# Go protobuf tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Smart Core tools
go install github.com/smart-core-os/sc-golang/cmd/protoc-gen-router@latest
go install github.com/smart-core-os/sc-golang/cmd/protoc-gen-wrapper@latest
```

Also required:
- `protoc` - Protocol Buffers compiler (https://grpc.io/docs/protoc-installation/)
- `protoc-gen-grpc-web` - JavaScript/TypeScript gRPC-Web plugin
- `yarn` - JavaScript package manager
