#!/usr/bin/env sh

export GOTOOLCHAIN=go1.21.13
GO_TAGS="dev"

# Build everything: core templates/queries first, then CLI, then plugins
(cp go.work.default go.work && \
        rm -rf **/*_templ.go && \
        sh -c "cd core && templ generate" && \
        ./scripts/sqlc-gen.sh ./core && \
        go run -tags="${GO_TAGS}" ./core/cmd/build-cli/main.go && \
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
    "hosts.json" \
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
