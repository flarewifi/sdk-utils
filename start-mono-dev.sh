#!/usr/bin/env bash

ROOT_DIR="$(pwd)"
DB_DRIVER="sqlite"
OS_CONFIG="wan-lan-mono"
GO_TAGS="dev mono $DB_DRIVER"
CREATE_PUGINS_INIT="./tools/cmd/make-mono/main.go"
SYNC_VERSION="./tools/cmd/sync-versions/main.go"
BUILD_ASSETS_MAIN="./tools/cmd/build-assets/main.go"
BUILD_MONO_BIN="./tools/cmd/create-mono-bin/main.go"
FLARE_CLI_MAIN="./core/internal/cli/main.go"

cp go.work.default go.work && \
    echo "Generating templ files..." && \
    rm -rf **/*_templ.go && \
    sh -c "cd core && templ generate" && \
    echo "Generating sqlc queires..." && \
    sh -c "./scripts/sqlc-gen.sh ./core $DB_DRIVER" && \
    cp ./core/internal/api/plugin-init_mono.default \
    ./core/internal/api/plugin-init_mono.go && \
    echo "Scanning translations..." && \
    go run -tags="${GO_TAGS}" ./tools/cmd/scan-translations --silent && \
    go run -tags="${GO_TAGS}" $SYNC_VERSION && \
    go run -tags="${GO_TAGS}" $BUILD_ASSETS_MAIN && \
    go run -tags="${GO_TAGS}" $FLARE_CLI_MAIN fix-workspace && \
    go run -tags="${GO_TAGS}" $FLARE_CLI_MAIN build-templates && \
    go run -tags="${GO_TAGS}" $CREATE_PUGINS_INIT && \
    echo "Building mono binary..." && \
    go run -tags="${GO_TAGS}" $BUILD_MONO_BIN


if [ $? != 0 ]; then
    echo "Failed to build core system!"
    exit 1
fi

MONO_BIN_OUT="$ROOT_DIR/output/mono-bin-files"
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
    "os_release.json" \
    ; do

    rm -rf $APP_DIR/$f && \
        ln -s $(pwd)/$f $APP_DIR/$f || (echo "Failed to link $f" && exit 1)
done

echo "Copying mono bin files to app directory..."
# Copy files from mono bin output
rsync -a $MONO_BIN_OUT/ $APP_DIR/
rsync -a $MONO_BIN_OUT/data/ $DATA_DIR/
mkdir -p $APP_DIR/.tmp
touch $APP_DIR/.tmp/.server-up
rm -rf $APP_DIR/data
ln -sf $DATA_DIR $APP_DIR/data

echo
echo "Starting Flare Hotspot Mono Dev Environment..."
sh -c "cd $APP_DIR && ./start.sh"
