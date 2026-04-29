#!/usr/bin/env bash
if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <node_version>" >&2
  exit 1
fi
NODE_VERSION="$1"

set -e

ARCH="$(uname -m)"
if [[ $ARCH == "x86_64" ]]; then
  NODE_ARCH="x64"
elif [[ $ARCH == "aarch64" ]]; then
  NODE_ARCH="arm64"
else
  echo "Unsupported architecture: $ARCH"
  exit 1
fi
TARBALL_URL="https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${NODE_ARCH}.tar.xz"
TARBALL_FILE="node.tar.xz"
echo "Downloading Node.js from $TARBALL_URL..."
curl -L -o "$TARBALL_FILE" "$TARBALL_URL"

echo "Extracting Node.js..."
INSTALL_DIR="/opt/node"
mkdir -p "$INSTALL_DIR"
tar -xf "$TARBALL_FILE" -C "$INSTALL_DIR" --strip-components=1
rm "$TARBALL_FILE"

echo "Creating symlinks for node, npm, and npx..."
ln -s "$INSTALL_DIR/bin/node" /usr/local/bin/node
ln -s "$INSTALL_DIR/bin/npm" /usr/local/bin/npm
ln -s "$INSTALL_DIR/bin/npx" /usr/local/bin/npx

echo "Installing Yarn..."
npm install -g yarn

echo "Installation complete."