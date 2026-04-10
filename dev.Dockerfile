# Defines a development environment for working on sc-bos.
#
# Example invocation:
#   podman build -t sc-bos-dev -f dev.Dockerfile .
#   podman run -it --rm -p 8443:8443 -p 23557:23557 \
#     -v $(pwd):/src \
#     sc-bos-dev

FROM golang:1.26-trixie

WORKDIR /tmp/install

# Install CLI dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      build-essential \
      curl \
      git \
      unzip \
      wget \
      ca-certificates \
      xz-utils && \
    rm -rf /var/lib/apt/lists/*

RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0

# Install Node.js and Yarn
COPY scripts/dev-container/install-node.sh install-node.sh
RUN chmod +x install-node.sh && \
    ./install-node.sh 22.22.2

# Install protobuf tools
COPY scripts/dev-container/install-protoc.sh install-protoc.sh
RUN chmod +x install-protoc.sh && ./install-protoc.sh --protoc 34.0
RUN ./install-protoc.sh --protoc-gen-js 4.0.1
RUN ./install-protoc.sh --protoc-gen-grpc-web 2.0.2
RUN ./install-protoc.sh --protoc-gen-go 1.36.11
RUN ./install-protoc.sh --protoc-gen-go-grpc 1.5.1

WORKDIR /src
