#!/bin/bash

# Link the files into the runtime/current directory
# so that it can be deleted on software update testing without losing the files.

rm -rf ./runtime/current
mkdir -p ./runtime/current/plugins

for f in \
    "bin" \
    "core" \
    "main" \
    "sdk" \
    "node_modules" \
    "go.work" \
    ".go-version" \
    "start-dev.sh" \
    "data" \
    ; do

    sh -c "cd ./runtime/current/ && ln -s ../../$f $f"
done

sh -c "cd ./runtime/current/plugins/ && ln -s ../../../plugins/system ./system"
