#!/bin/sh

DB_DRIVER="sqlite"
GO_TAGS="prod $DB_DRIVER"
BUILD_CORE_MAIN="./tools/cmd/build-core"
BUILD_CLI_MAIN="./tools/cmd/build-cli"
BUILD_ASSETS_MAIN="./tools/cmd/build-assets"
FLARE_CLI_MAIN="./core/internal/cli"
PATH="$PATH:$HOME/go/bin"

cp go.work.default go.work && \
    (cd scripts && ./install-tools.sh) && \
    echo "Generating templ files..." && \
    rm -rf **/*_templ.go && \
    sh -c "cd ./core && templ generate" && \
    echo "Generating sqlc queires..." && \
    sh -c "./scripts/sqlc-gen.sh ./core $DB_DRIVER" && \
    go run -tags="${GO_TAGS}" $FLARE_CLI_MAIN fix-workspace && \
    go run -tags="${GO_TAGS}" $FLARE_CLI_MAIN build-templates && \
    go run -tags="${GO_TAGS}" $BUILD_ASSETS_MAIN && \
    go run -tags="${GO_TAGS}" $BUILD_CORE_MAIN && \
    go run -tags="${GO_TAGS}" $BUILD_CLI_MAIN && \
    ./bin/flare server
