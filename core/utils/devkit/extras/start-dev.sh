#!/usr/bin/env sh

GO_TAGS="dev"

(cp go.work.default go.work && \
        rm -rf **/*_templ.go && \
        ./bin/flare fix-workspace && \
        ./bin/flare build-plugins
) || (echo "Build failed" && exit 1)

APP_DIR="/opt/flarehotspot/app"
DATA_DIR="/opt/flarehotspot/data"
rm -rf $APP_DIR/*
mkdir -p $APP_DIR

for f in \
    "bin" \
    "core" \
    "defaults" \
    "plugins" \
    "sdk" \
    "scripts" \
    "go.work" \
    "go.sum" \
    "start.sh" \
    ; do

    rm -rf $APP_DIR/$f && \
    ln -s $(pwd)/$f $APP_DIR/$f || (echo "Failed to link $f" && exit 1)
done

mkdir -p $APP_DIR/.tmp
touch $APP_DIR/.tmp/.server-up
sh -c "cd $APP_DIR && ./bin/flare server"
