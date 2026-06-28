#!/usr/bin/env sh

# Devkit dev loop. Runs entirely from /app (the mounted devkit root) — NOT the
# production-style /opt/flarewifi/app symlink layout. The prebuilt flare CLI
# resolves getRootDir() to the CWD, so running here makes PathDataDir=/app/data,
# where the developer's plugins (data/plugins/devel/*) and the shipped Devkit
# theme resources (plugins/installed/com.flarego.devkit) live. The /opt/flarewifi
# symlink dance is production-image fidelity the devkit doesn't need.
#
# Plugins are compiled by the prebuilt flare CLI, which was built with
# `dev devkit sqlite`; flare propagates those exact tags to every plugin it builds
# (tags.GetBuildTags reflects the running binary's compile tags). 3rd-party plugins
# compile against the SDK alone — the closed-source core ships only as binaries.

# Activate the arch-specific binaries (bin/flare, bin/livereload, core/plugin.so).
# Idempotent: docker-cmd.sh already runs this, but start-dev.sh may be run directly.
sh ./select-arch.sh

(cp go.work.default go.work && \
        rm -rf **/*_templ.go && \
        ./bin/flare fix-workspace && \
        ./bin/flare build-plugins \
) || { echo "Build failed"; exit 1; }

mkdir -p .tmp
touch .tmp/.server-up

# Devel plugins (the developer's, under data/plugins/devel) are compiled by
# `flare server` at boot, from this CWD (/app).
./bin/flare server
