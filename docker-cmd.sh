#!/bin/sh

cp go.work.default go.work && \
    reflex \
    -r '\.(go|templ|sql|js|css|json)$' \
    -R '(plugin|package).json$' \
    -R '_templ\.go$' \
    -R '\.tmp\/.*' \
    -R '^output\/.*' \
    -R '^bin\/.*' \
    -R 'db\/queries\/.*' \
    -R 'node_modules' \
    -R 'data\/config\/.*' \
    -R 'resources\/assets\/dist' \
    -R 'storage\/.*' \
    -R 'plugins\/installed\/.*' \
    -R 'plugins\/backups\/.*' \
    -R 'plugins\/updates\/.*' \
    -s -- sh -c './start-dev.sh' -v &

touch "/tmp/.flare-up" && \
    go run -tags="dev" ./core/cmd/livereload/main.go &

wait
