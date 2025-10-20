#!/usr/bin/env sh

BUILD_TAGS="dev"
BUILD_CORE_MAIN="./core/cmd/build-core"
BUILD_CLI_MAIN="./core/cmd/build-cli"
FLARE_CLI_MAIN="./core/internal/cli"
SYNC_VERSION="./core/cmd/sync-versions/main.go"
FLARE_BIN="./bin/flare"

(cp go.work.default go.work && \
        echo "Cleaning templ output files..." && \
        rm -rf **/*_templ.go && \
        rm -rf core/internal/db/sqlc && \
        sh -c "cd core/resources/views && templ generate" && \
        go run -tags="${BUILD_TAGS}" $SYNC_VERSION && \
        go run -tags="${BUILD_TAGS}" $FLARE_CLI_MAIN fix-workspace && \
        go run -tags="${BUILD_TAGS}" $FLARE_CLI_MAIN build-templates && \
        go run -tags="${BUILD_TAGS}" $BUILD_CORE_MAIN && \
        go run -tags="${BUILD_TAGS}" $BUILD_CLI_MAIN
) || (echo "Build failed" && exit 1)

if [ $? != 0 ]; then
    echo "Failed to build core system!"
    exit 1
fi

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

    rm -rf $APP_DIR/$f && \
        ln -s $(pwd)/$f $APP_DIR/$f || (echo "Failed to link $f" && exit 1)
done

mkdir -p $APP_DIR/.tmp
touch $APP_DIR/.tmp/.server-up
sh -c "cd $APP_DIR && ./start.sh"
