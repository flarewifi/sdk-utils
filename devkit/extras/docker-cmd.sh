#!/bin/sh

# Devkit entrypoint. Unlike the in-repo docker-cmd.sh, this runs in a
# closed-source devkit where no core Go source ships: the live-reload server is a
# precompiled binary (bin/livereload) instead of `go run ./core/cmd/livereload`,
# and core/product.json is shipped prebuilt instead of generated via
# `go run ./core/cmd/gen-product-version`.

START_SH=$1

# Activate the binaries (bin/flare, bin/livereload, core/plugin.so) matching this
# container's CPU architecture before anything launches them. reflex ignores bin/
# (-R '^bin\/.*') and .so files, so the selector's writes never trigger a rebuild.
sh ./select-arch.sh

# Setup go.work first (needed by both reflex and the plugin builds)
cp go.work.default go.work

# Start file watcher: rebuild the developer's plugins on source changes. There is
# no core source here, so only plugin files (under data/plugins/devel) trigger a
# rebuild of the running server via $START_SH.
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
    -R 'plugins\/installed\/.*' \
    -R 'plugins\/backups\/.*' \
    -R 'plugins\/updates\/.*' \
    -R 'plugin\-init_mono\.(go|default)$' \
    -R 'system\-plugins\-init\.(go|default)$' \
    -s -- sh -c "$START_SH" -v &

# Start the precompiled live-reload server (go.work is now available)
touch "/tmp/.flare-up" && \
    ./bin/livereload &

wait
