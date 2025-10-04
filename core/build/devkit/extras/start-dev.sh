#!/usr/bin/env sh

(cp go.work.default go.work && \
        rm -rf **/*_templ.go && \
        ./bin/flare fix-workspace && \
        ./bin/flare build-plugins
) || (echo "Build failed" && exit 1)

RUNTIME_DIR="/etc/flarehotspot"
DATA_DIR="/var/lib/flarehotspot"
rm -rf $RUNTIME_DIR/*

for f in \
    "bin" \
    "core" \
    "defaults" \
    "main" \
    "node_modules" \
    "plugins" \
    "sdk" \
    "go.work" \
    "go.sum" \
    "start.sh" \
    ; do

    ln -s $(pwd)/$f $RUNTIME_DIR/$f || (echo "Failed to link $f" && exit 1)
done

sh -c "cd $RUNTIME_DIR && ./bin/flare server"
