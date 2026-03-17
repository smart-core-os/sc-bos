#!/bin/bash

# Usage: publish-all.sh 1.0.0-beta.25
# Note, don't include the v prefix on the version

set -e

pub_prepare() {
  echo $1/publish.sh prepare "$2"
  pushd $1
  ./publish.sh prepare "$2"
  popd
}

pub_perform() {
  echo $1/publish.sh perform "$2"
  pushd $1
  ./publish.sh perform "$2"
  popd
}

pub_prepare "go" "$1"
pub_prepare "grpc-web" "$1"
pub_prepare "java" "$1"
pub_prepare "node" "$1"

git commit -m "chore: publish v$1" --no-status

pub_perform "go" "$1"
pub_perform "grpc-web" "$1"
pub_perform "java" "$1"
pub_perform "node" "$1"

git tag "v$1"
echo "sc-api $1 has been published"
echo "Don't forget to git push --tags"
