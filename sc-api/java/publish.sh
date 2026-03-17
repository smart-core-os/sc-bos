#!/bin/bash

set -e

if [ $1 = "prepare" ]; then
  echo "mvn versions:set -DnewVersion=$2"
  mvn versions:set -DnewVersion=$2 -DgenerateBackupPoms=false
  git add pom.xml
fi

if [ $1 = "perform" ]; then
  echo "mvn deploy"
  mvn deploy
fi

