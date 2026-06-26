#!/usr/bin/env sh

export GOTOOLCHAIN=go1.21.13
GO_TAGS="dev"

# Build everything: core templates/queries first, then CLI, then plugins
(cp go.work.default go.work && \
        rm -rf **/*_templ.go && \
        sh -c "cd core && templ generate" && \
        ./scripts/sqlc-gen.sh ./core && \
        # Generate static system plugin loader (writes core/internal/api/system-plugins-init.go
        # and <plugin>/static/main.go for every system plugin). Must run before
        # any compile of the api package — otherwise LoadSystemPlugins resolves
        # to the no-op .default stub and any system plugins are silently absent.
        go run -tags="${GO_TAGS}" ./core/cmd/sysplugin-prepare/main.go && \
        go run -tags="${GO_TAGS}" ./core/cmd/build-cli/main.go && \
        ./bin/flare fix-workspace && \
        # Generate core/product.json from the core/plugin.json version. The
        # software-release build stamps product.json with the per-partner product
        # version; dev has no such stamp, so generate a stand-in (== core version)
        # each reload. product.json is gitignored, and reflex ignores it (see
        # docker-cmd.sh) so writing it does not retrigger a rebuild loop.
        go run -tags="${GO_TAGS}" ./core/cmd/gen-product-version/main.go && \
        # Rebuild core assets (resources/assets/dist) on every reload. `flare
        # build-plugins` below only rebuilds plugin assets, so without this a core
        # JS/CSS edit would never reach the browser. --core-only keeps the hot
        # loop fast (plugins are handled by build-plugins).
        go run -tags="${GO_TAGS}" ./core/cmd/build-assets/main.go --core-only
) || { echo "Build failed"; exit 1; }

APP_DIR="/opt/flarewifi/app"
DATA_DIR="/opt/flarewifi/data"

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
    ln -s $(pwd)/$f $APP_DIR/$f || { echo "Failed to link $f"; exit 1; }
done

# Create temp directory marker
mkdir -p $APP_DIR/.tmp
touch $APP_DIR/.tmp/.server-up

# Ensure system/local plugins are installed under runtime APP_DIR
APP_DIR="$APP_DIR" APP_TMP="$APP_TMP" ./bin/flare build-plugins || { echo "Build plugins failed"; exit 1; }

# Dev mode: prevent stale update tarballs from wiping app symlinks
if [ -d "/opt/flarewifi/data/storage/system/updates" ]; then
    rm -rf /opt/flarewifi/data/storage/system/updates/*.tar.gz
    rm -rf /opt/flarewifi/data/storage/system/updates/.dl_software_update_complete
fi


# Run start.sh from the app directory
cd $APP_DIR && ./start.sh
