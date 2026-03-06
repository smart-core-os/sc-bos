#!/bin/bash

set -e

if [ $1 = "perform" ]; then
  echo "git tag go/$2"
  git tag "go/v$2"
fi
