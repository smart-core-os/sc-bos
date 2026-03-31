#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR=$(git rev-parse --show-toplevel)
go run $ROOT_DIR/cmd/tools/genproto
