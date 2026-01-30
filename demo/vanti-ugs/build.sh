#!/usr/bin/env bash

set -euo pipefail

# Builds the sc-bos docker image, which includes the sc-bos binary and ops-ui web app.
# Then extends that image with demo configuration and opinionated defaults.

containerCmd="podman"

if ! command -v $containerCmd &> /dev/null; then
  echo "Error: $containerCmd is not installed."
  exit 1
fi

# The sc-bos build requires secrets that are usually in the users home .npmrc file,
# so make sure it exists.
if [ ! -f "$HOME/.npmrc" ]; then
  echo "Error: $HOME/.npmrc file not found. Please create it with your npm credentials."
  exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
GIT_VERSION=$(git describe --tags --always)
BASE_IMAGE="localhost/smart-core-os-sc-bos:demo-ugs-base"
PLATFORMS="linux/amd64,linux/arm64"

cd $REPO_ROOT
echo "Preparing $containerCmd for building the image..."
for name in "localhost/demo-ugs-sc-bos" "localhost/demo-ugs-seed-db"; do
  if $containerCmd manifest exists "$name"; then
    $containerCmd manifest rm "$name"
  fi
  $containerCmd manifest create "$name"
done

# Build using the shared Dockerfile in the project root directory
#echo "Building the sc-bos base image with version: $GIT_VERSION"
#$containerCmd build \
#  --platform=$PLATFORMS \
#  --build-arg GIT_VERSION=$GIT_VERSION \
#  --secret=id=npmrc,src=$HOME/.npmrc \
#  --manifest "$BASE_IMAGE" \
#  -f $REPO_ROOT/Dockerfile \
#  .

# Add config to the sc-bos image
echo "Building the sc-bos demo image with version: $GIT_VERSION"
$containerCmd build \
  --build-arg GIT_VERSION=$GIT_VERSION \
  --secret=id=npmrc,src=$HOME/.npmrc \
  --platform=$PLATFORMS \
  --manifest "demo-ugs-sc-bos" \
  -f demo/vanti-ugs/Dockerfile-Ugs .
# Build the db seeder image
echo "Building the sc-bos demo seed-db image with version: $GIT_VERSION"
$containerCmd build \
  --platform=$PLATFORMS \
  --manifest "demo-ugs-seed-db" \
  -f demo/vanti-ugs/Dockerfile-SeedDb .

TAG_VERSION=$(basename $GIT_VERSION)
echo "Push the images to the container registry using:"
echo "$containerCmd manifest push --all demo-ugs-sc-bos docker://ghcr.io/smart-core-os/sc-bos/demo-ugs-sc-bos:$TAG_VERSION"
echo "$containerCmd manifest push --all demo-ugs-seed-db docker://ghcr.io/smart-core-os/sc-bos/demo-ugs-seed-db:$TAG_VERSION"
