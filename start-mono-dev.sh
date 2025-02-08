#!/usr/bin/env bash

BUILD_TAGS="dev mono"
CREATE_MONO="./core/cmd/make-mono/main.go"
MONO_SERVER="./core/cmd/mono-server/main.go"
CLI_MAIN="./core/internal/cli/main.go"

cp go.work.default go.work && \
    rm -rf **/*_templ.go && \
    rm -rf core/internal/db/sqlc && \
    go run -tags="${BUILD_TAGS}" $LINK_NODE_MODULES && \
    go run -tags="${BUILD_TAGS}" $CLI_MAIN fix-workspace && \
    go run -tags="${BUILD_TAGS}" $CLI_MAIN build-templates && \
    go run -tags="${BUILD_TAGS}" $CREATE_MONO && \
    go run -tags="${BUILD_TAGS}" $MONO_SERVER
