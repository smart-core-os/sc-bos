# Golang Smart core API library
This is the (generated) source of the Go packages for working with Smart Core APIs.

## Using
To install the packages on your system, you should use:
```shell script
$ go get -u github.com/smart-core-os/sc-api/go
```

In your Go files you can then use
```go
import "github.com/smart-core-os/sc-api/go/"
```

## Updating

### Prerequisites

You will need `protoc-gen-go` to generate Go code, see [go-generated](https://developers.google.com/protocol-buffers/docs/reference/go-generated#invocation).
To install it run:
```
go install google.golang.org/protobuf/cmd/protoc-gen-go
```

This will install into your `$GOBIN` the required protoc compiler plugin to generate the `--go-grpc_out` files.

### Generating the source

If you have made changes to the API definition files and need to re-generate one or more packages, you'll need to run
the following from the root of this folder (i.e. `/go`):
```shell script
$ protoc -I ../protobuf --go_out=paths=source_relative:./ --go-grpc_out=paths=source_relative:./ ../protobuf/<package>/*.proto
```
(**Note:** the wildcard syntax will only work on Linux - from Windows you'll need to specify each proto individually)

Then run the following to build & test the Go module:
```shell script
$ go build ./... && go test ./...
```

For convenience, a shell script is included to regenerate all packages:
```shell script
$ ./generate.sh
```

If you have added a new package, please add it to the script above.

### Deploying a new release
// TODO: details for versioning, etc. In the meantime, see: 
https://github.com/golang/go/wiki/Modules#how-to-prepare-for-a-release
