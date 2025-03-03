#!/usr/bin/env ash

BUILD_TAGS="prod mono"
CREATE_MONO="./core/cmd/make-mono/main.go"
MONO_SERVER="./core/cmd/mono-server/main.go"

cp go.work.default go.work && \
    go run -tags="${BUILD_TAGS}" $CREATE_MONO && \
    go run -tags="${BUILD_TAGS}" $MONO_SERVER
