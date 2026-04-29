#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat >&2 << EOF
Usage: $0
  --protoc <protoc_version>
  --protoc-gen-js <protoc-gen-js_version>
  --protoc-gen-grpc-web <protoc-gen-grpc-web_version>
  --protoc-gen-go <protoc-gen-go_version>
  --protoc-gen-go-grpc <protoc-gen-go-grpc_version>
EOF
  exit 1
}

args=$(getopt -a -o '' --long 'protoc:,protoc-gen-js:,protoc-gen-grpc-web:,protoc-gen-go:,protoc-gen-go-grpc:' -- "$@")
if [[ $? -ne 0 ]]; then
  usage
fi

eval set -- "$args"
while :
do
  case "$1" in
    --protoc)               PROTOC_VERSION="$2";              shift 2 ;;
    --protoc-gen-js)        PROTOC_GEN_JS_VERSION="$2";       shift 2 ;;
    --protoc-gen-grpc-web)  PROTOC_GEN_GRPC_WEB_VERSION="$2"; shift 2 ;;
    --protoc-gen-go)        PROTOC_GEN_GO_VERSION="$2";       shift 2 ;;
    --protoc-gen-go-grpc)   PROTOC_GEN_GO_GRPC_VERSION="$2";  shift 2 ;;
    --) shift; break ;; # end of options
    *) echo "Error: unsupported option: $1" && usage ;;
  esac
done

if [[ -z "${PROTOC_VERSION:-}" && -z "${PROTOC_GEN_JS_VERSION:-}" && -z "${PROTOC_GEN_GRPC_WEB_VERSION:-}" \
     && -z "${PROTOC_GEN_GO_VERSION:-}" && -z "${PROTOC_GEN_GO_GRPC_VERSION:-}" ]]; then
  echo "Error: at least one option is required." >&2
  usage
fi

ARCH="$(uname -m)"
case $ARCH in
  x86_64) PROTOC_ARCH="x86_64" ;;
  aarch64) PROTOC_ARCH="aarch_64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

# Download and install protoc
if [[ -n "${PROTOC_VERSION:-}" ]]; then
  PROTOC_URL="https://github.com/protocolbuffers/protobuf/releases/download/v$PROTOC_VERSION/protoc-$PROTOC_VERSION-linux-$PROTOC_ARCH.zip"
  PROTOC_FILE="protoc.zip"
  PROTOC_INSTALL="/opt/protoc"
  echo "Downloading protoc from $PROTOC_URL..."
  curl -L -o "$PROTOC_FILE" "$PROTOC_URL"
  echo "Extracting protoc..."
  mkdir -p "$PROTOC_INSTALL"
  unzip -o "$PROTOC_FILE" -d "$PROTOC_INSTALL"
  rm "$PROTOC_FILE"
  ln -s "$PROTOC_INSTALL/bin/protoc" /usr/local/bin/protoc
fi

# Download and install protoc-gen-js
if [[ -n "${PROTOC_GEN_JS_VERSION:-}" ]]; then
  PROTOC_GEN_JS_URL="https://github.com/protocolbuffers/protobuf-javascript/releases/download/v$PROTOC_GEN_JS_VERSION/protobuf-javascript-$PROTOC_GEN_JS_VERSION-linux-$PROTOC_ARCH.tar.gz"
  PROTOC_GEN_JS_FILE="protoc-gen-js.tar.gz"
  PROTOC_GEN_JS_INSTALL="/opt/protoc-gen-js"
  echo "Downloading protoc-gen-js from $PROTOC_GEN_JS_URL..."
  curl -L -o "$PROTOC_GEN_JS_FILE" "$PROTOC_GEN_JS_URL"
  echo "Extracting protoc-gen-js..."
  mkdir -p "$PROTOC_GEN_JS_INSTALL"
  tar -xzf "$PROTOC_GEN_JS_FILE" -C "$PROTOC_GEN_JS_INSTALL" --strip-components=1
  rm "$PROTOC_GEN_JS_FILE"
  ln -s "$PROTOC_GEN_JS_INSTALL/protoc-gen-js" /usr/local/bin/protoc-gen-js
fi

# Download and install protoc-gen-grpc-web
if [[ -n "${PROTOC_GEN_GRPC_WEB_VERSION:-}" ]]; then
  PROTOC_GEN_GRPC_WEB_URL="https://github.com/grpc/grpc-web/releases/download/$PROTOC_GEN_GRPC_WEB_VERSION/protoc-gen-grpc-web-$PROTOC_GEN_GRPC_WEB_VERSION-linux-$ARCH"
  # single file, install directly in place
  PROTOC_GEN_GRPC_WEB_INSTALL="/usr/local/bin/protoc-gen-grpc-web"
  echo "Downloading protoc-gen-grpc-web from $PROTOC_GEN_GRPC_WEB_URL..."
  curl -L -o "$PROTOC_GEN_GRPC_WEB_INSTALL" "$PROTOC_GEN_GRPC_WEB_URL"
  chmod +x "$PROTOC_GEN_GRPC_WEB_INSTALL"
fi

# Download and install protoc-gen-go and protoc-gen-go-grpc
if [[ -n "${PROTOC_GEN_GO_VERSION:-}" ]]; then
  go install "google.golang.org/protobuf/cmd/protoc-gen-go@v$PROTOC_GEN_GO_VERSION"
fi
if [[ -n "${PROTOC_GEN_GO_GRPC_VERSION:-}" ]]; then
  go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@v$PROTOC_GEN_GO_GRPC_VERSION"
fi