#!/bin/sh

cp go.work.default go.work && \
    reflex \
    -r '\.(go|templ|sql|js|css|json)$' \
    -R '(plugin|package|package\-lock).json$' \
    -R '_templ\.go$' \
    -R '\.tmp\/.*' \
    -R '^output\/.*' \
    -R '^bin\/.*' \
    -R 'db\/queries\/.*' \
    -R 'node_modules\/.*' \
    -R 'data\/config\/.*' \
    -R 'data\/storage\/.*' \
    -R 'resources\/assets\/dist\/.*' \
    -R 'plugins\/installed\/.*' \
    -R 'plugins\/backups\/.*' \
    -R 'plugins\/updates\/.*' \
    -s -- sh -c './start-dev.sh' -v &

touch "/tmp/.flare-up" && \
    go run -tags="dev" ./core/cmd/livereload/main.go &

wait
