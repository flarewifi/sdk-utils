#!/bin/bash

rm -rf ./runtime/current
mkdir -p ./runtime/current

for f in \
    "bin" \
    "core" \
    "main" \
    "sdk" \
    "node_modules" \
    "go.work" \
    ".go-version" \
    "start-dev.sh" \
    "shared" \
    ; do

    sh -c "cd ./runtime/current/ && ln -s ../../$f $f"
done

