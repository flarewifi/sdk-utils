#!/bin/sh

DB_DRIVER="sqlite"
GO_TAGS="dev $DB_DRIVER"

cp go.work.default go.work && \
    echo "Generating templ files..." && \
    rm -rf **/*_templ.go && \
    sh -c "cd core && templ generate" && \
    echo "Generating sqlc queires..." && \
    sh -c "./scripts/sqlc-gen.sh ./core $DB_DRIVER" && \
		go run --tags="$GO_TAGS" ./tools/cmd/create-devkit/main.go
