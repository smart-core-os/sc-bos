#!/bin/bash

set -e

function gen_folder() {
  echo $1/generate.sh
  pushd $1
  ./generate.sh
  popd
}

gen_folder "go"
gen_folder "grpc-web"
gen_folder "java"
gen_folder "node"
