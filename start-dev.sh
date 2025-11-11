#!/usr/bin/env sh

DB_DRIVER="postgres"
GO_TAGS="dev $DB_DRIVER"
OS_CONFIG="wan-lan"
BUILD_CORE_MAIN="./tools/cmd/build-core"
BUILD_CLI_MAIN="./tools/cmd/build-cli"
BUILD_ASSETS_MAIN="./tools/cmd/build-assets"
SYNC_VERSION="./tools/cmd/sync-versions/main.go"
FLARE_CLI_MAIN="./core/internal/cli"
FLARE_BIN="./bin/flare"

cp go.work.default go.work && \
    echo "Generating templ files..." && \
    rm -rf **/*_templ.go && \
    sh -c "cd core && templ generate" && \
    echo "Generating sqlc queires..." && \
    sh -c "./scripts/sqlc-gen.sh ./core $DB_DRIVER" && \
    go run -tags="${GO_TAGS}" $SYNC_VERSION && \
    go run -tags="${GO_TAGS}" $BUILD_ASSETS_MAIN && \
    go run -tags="${GO_TAGS}" $FLARE_CLI_MAIN fix-workspace && \
    go run -tags="${GO_TAGS}" $FLARE_CLI_MAIN build-templates && \
    go run -tags="${GO_TAGS}" $BUILD_CORE_MAIN && \
    go run -tags="${GO_TAGS}" $BUILD_CLI_MAIN

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
    "plugins" \
    "sdk" \
    "scripts" \
    "tools" \
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
