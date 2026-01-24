#!/bin/sh

START_SH=$1

# Setup go.work first (needed by both reflex and livereload)
cp go.work.default go.work

# Start file watcher for rebuilds
reflex \
    -r '\.(go|templ|sql|js|css|json|sh)$' \
    -R '(plugin|package|package\-lock).json$' \
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
    -s -- sh -c "$START_SH" -v &

# Start livereload server (go.work is now available)
touch "/tmp/.flare-up" && \
    go run -tags="dev" ./core/cmd/livereload/main.go &

wait
