#!/bin/bash

set -e

rm -rf ./proto/info ./proto/traits ./proto/types
yarn run gen
