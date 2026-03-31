#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
GO_PATH=$(go tool -n protoc-gen-go)
GO_GRPC_PATH=$(go tool -n protoc-gen-go-grpc)
WRAPPER_PATH=$(go tool -n protoc-gen-wrapper)
ROUTER_PATH=$(go tool -n protoc-gen-router)
protoc \
  -I=$REPO_ROOT \
  --plugin=protoc-gen-go=$GO_PATH \
  --go_out=paths=source_relative:$REPO_ROOT \
  --plugin=protoc-gen-go-grpc=$GO_GRPC_PATH \
  --go-grpc_out=paths=source_relative:$REPO_ROOT \
  --plugin=protoc-gen-wrapper=$WRAPPER_PATH \
  --wrapper_opt=usePaths=true \
  --wrapper_out=paths=source_relative:$REPO_ROOT \
  --plugin=protoc-gen-router=$ROUTER_PATH \
  --router_opt=usePaths=true \
  --router_out=paths=source_relative:$REPO_ROOT \
  pkg/driver/bacnet/rpc/bacnet.proto
