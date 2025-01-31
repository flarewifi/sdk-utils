#!/usr/bin/env sh

cp go.work.default go.work
rm -rf **/*_templ.go
./bin/flare fix-workspace
./bin/flare build-plugins
./bin/flare server
