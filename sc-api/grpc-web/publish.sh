#!/bin/bash

set -e

if [ $1 = "prepare" ]; then
  echo "yarn version --new-version $2 --no-git-tag-version"
  yarn version --new-version "$2" --no-git-tag-version
  git add package.json
fi

if [ $1 = "perform" ]; then
  echo "yarn publish"
  yarn publish --non-interactive
fi
