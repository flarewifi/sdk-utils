#!/usr/bin/env sh

GO_TAGS="dev"

# Build everything
(cp go.work.default go.work && \
        rm -rf **/*_templ.go && \
        ./bin/flare fix-workspace && \
        ./bin/flare build-plugins
) || (echo "Build failed" && exit 1)

APP_DIR="/opt/flarehotspot/app"
DATA_DIR="/opt/flarehotspot/data"

# Clean and prepare app directory
rm -rf $APP_DIR/*
mkdir -p $APP_DIR

# Create symlinks to emulate production structure
for f in \
    "bin" \
    "core" \
    "defaults" \
    "plugins" \
    "sdk" \
    "scripts" \
    "go.work" \
    "go.sum" \
    "start.sh" \
    ; do

    rm -rf $APP_DIR/$f && \
    ln -s $(pwd)/$f $APP_DIR/$f || (echo "Failed to link $f" && exit 1)
done

# Create temp directory marker
mkdir -p $APP_DIR/.tmp
touch $APP_DIR/.tmp/.server-up

# Run start.sh from the app directory
cd $APP_DIR && ./start.sh
