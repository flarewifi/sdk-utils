#!/bin/sh

echo "Installing CLI tools..."
go install github.com/cespare/reflex@v0.3.1 && \
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0 && \
    (cd ./sdk/libs/templ-0.2.793 && go install ./cmd/templ)
