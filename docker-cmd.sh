#!/bin/sh

START_SH=$1

# Setup go.work first (needed by both reflex and livereload)
cp go.work.default go.work

# Generate core/product.json (version copied from core/plugin.json) up front, so
# it exists before the first reflex build and for every dev entrypoint
# ($START_SH = start-dev.sh / start-mono-dev.sh / start-openwrt-dev.sh). The start
# script regenerates it on each reload to track plugin.json version bumps mid-
# session. product.json is gitignored and excluded from the reflex watcher below,
# so writing it never retriggers a rebuild. Non-fatal: product.Version() falls back
# to the core version when product.json is absent.
go run -tags="dev" ./core/cmd/gen-product-version/main.go || echo "warning: failed to generate core/product.json"

# Start file watcher for rebuilds
reflex \
    -r '\.(go|templ|sql|js|css|json|sh)$' \
    -R '(plugin|product|package|package\-lock).json$' \
    -R '_templ\.go$' \
    -R '\.tmp\/.*' \
    -R '^output\/.*' \
    -R '^bin\/.*' \
    -R 'os_release\.json$' \
    -R 'db\/queries\/.*' \
    -R 'node_modules\/.*' \
    -R 'data\/config\/.*' \
    -R 'data\/storage\/.*' \
    -R 'resources\/assets\/dist\/.*' \
    -R 'resources\/translations\/.*\.json$' \
    -R 'plugins\/installed\/.*' \
    -R 'plugins\/backups\/.*' \
    -R 'plugins\/updates\/.*' \
    -R 'plugin\-init_mono\.(go|default)$' \
    -R 'system\-plugins\-init\.(go|default)$' \
    -s -- sh -c "$START_SH" -v &

# Start livereload server (go.work is now available)
touch "/tmp/.flare-up" && \
    go run -tags="dev" ./core/cmd/livereload/main.go &

wait
