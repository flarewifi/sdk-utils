#!/usr/bin/env sh

BUILD_TAGS="dev"
BUILD_CORE_MAIN="./core/cmd/build-core"
BUILD_CLI_MAIN="./core/cmd/build-cli"
SYNC_VERSION="./core/cmd/sync-versions/main.go"
LINK_NODE_MODULES="./core/cmd/link-node-modules"
FLARE_BIN="./bin/flare"

(cp go.work.default go.work && \
        rm -rf **/*_templ.go && \
        rm -rf core/internal/db/sqlc && \
        go run -tags="${BUILD_TAGS}" $LINK_NODE_MODULES && \
        go run -tags="${BUILD_TAGS}" $BUILD_CLI_MAIN && \
        go run -tags="${BUILD_TAGS}" $SYNC_VERSION && \
        sh -c "$FLARE_BIN fix-workspace" && \
        sh -c "$FLARE_BIN build-templates" && \
        go run -tags="${BUILD_TAGS}" $BUILD_CORE_MAIN
) || (echo "Build failed" && exit 1)

APP_DIR="/opt/flarehotspot/app"
DATA_DIR="/opt/flarehotspot/data"
rm -rf $APP_DIR/*
mkdir -p $APP_DIR

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

    ln -s $(pwd)/$f $APP_DIR/$f || (echo "Failed to link $f" && exit 1)
done

sh -c "cd $APP_DIR && ./start.sh"
