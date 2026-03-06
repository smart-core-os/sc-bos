#!/bin/bash

set -e

rm -rf info traits types
protoc -I ../protobuf --js_out=import_style=commonjs,binary:. \
  --grpc-web_out=import_style=commonjs+dts,mode=grpcwebtext:. \
  ../protobuf/info/*.proto \
  ../protobuf/traits/*.proto \
  ../protobuf/types/*.proto \
  ../protobuf/types/time/*.proto
