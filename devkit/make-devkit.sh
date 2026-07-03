#!/bin/sh

DB_DRIVER="sqlite"
# The devkit core + flare CLI are built with the additional `devkit` tag (on top
# of `dev`). Because tags.GetBuildTags() reflects the running binary's compile
# tags, the resulting flare CLI propagates `dev devkit sqlite` to every plugin it
# builds inside the devkit runtime.
GO_TAGS="dev devkit $DB_DRIVER"

# The Devkit theme system plugin is NOT committed in this repo. It is cloned from
# its own GitHub repo into data/plugins/system/com.flarego.devkit by the
# clone-devkit-plugin step below (core/utils/clone-plugins.go), BEFORE any step
# that scans data/plugins/system. sysplugin-prepare then statically links every
# plugin found there into the devkit's flare CLI / core — straight from source,
# with no staging. (For a plain local-dev `start-dev.sh` run you clone the plugin
# into that same location yourself to link it and iterate without a devkit rebuild.)
#
# Local-dev-only plugins that must NOT ship in the devkit live under
# data/plugins/devel instead. Devel plugins are compiled at runtime by the flare
# CLI and are never copied into the release nor static-linked, so they stay out of
# the distribution automatically — no exclusion step is needed here.
#
# (The developer upload/install panel used to live here as com.flarego.developer;
# it has since been folded into the com.flarego.devkit system plugin and now ships
# with the devkit itself.)

# Revert the transient `require`/`replace` entries link_system_plugin_modules adds
# to the protected core/go.mod. The plugin SOURCE is committed (never staged), so
# it is NOT removed. Deterministic and idempotent — `go mod edit -droprequire/-dropreplace`
# is a no-op (no churn) when the entry is absent, and is robust where a file-backup
# trap is not (SIGKILL, failed mktemp). Module names are read from the committed
# plugin sources under data/plugins/system.
revert_system_plugin_modules() {
    for d in data/plugins/system/*/; do
        [ -f "$d/go.mod" ] || continue
        mod="$(awk '/^module /{print $2; exit}' "$d/go.mod" 2>/dev/null)"
        [ -n "$mod" ] && ( cd core && go mod edit -droprequire="$mod" -dropreplace="$mod" 2>/dev/null || true )
    done
}

# Single EXIT cleanup: revert the transient go.mod edits.
cleanup_devkit_build() {
    revert_system_plugin_modules
}
trap cleanup_devkit_build EXIT

# Statically-linked system plugins are imported by the generated
# core/internal/api/system-plugins-init.go (com.flarego.<pkg>/system). go.work
# `use` alone resolves these for a plain `go build`, but the release build uses
# `-trimpath`, under which a bare require still fetches — only a path `replace`
# pins it, and (Go 1.21) the replace path string must match the go.work `use`
# path. So add `require <mod> v0.0.0` + `replace <mod> => ../data/plugins/system/<n>`
# (relative to core/, matching the repo-root go.work `use ./data/plugins/system/<n>`)
# for each system plugin. This is for the IN-PLACE builds (flare CLI, livereload,
# gen-product-version). The core.so build copies core/go.mod into an isolated
# workspace and rewrites this replace to ../../data/... itself (see BuildPluginSo).
# Reverted by the trap above.
link_system_plugin_modules() {
    for d in data/plugins/system/*/; do
        [ -f "$d/go.mod" ] || continue
        mod="$(awk '/^module /{print $2; exit}' "$d/go.mod")"
        [ -n "$mod" ] || continue
        rel="../${d%/}" # ../data/plugins/system/<n> (relative to core/)
        ( cd core && go mod edit -require="$mod@v0.0.0" -replace="$mod=$rel" )
    done
}

# Recover from a previous run killed with SIGKILL (the trap only fires on a clean
# EXIT): revert stale core/go.mod require/replace before re-staging below.
revert_system_plugin_modules

cp go.work.default go.work && \
    echo "Cloning Devkit theme system plugin from GitHub into data/plugins/system..." && \
    go run --tags="$GO_TAGS" ./core/cmd/clone-devkit-plugin/main.go && \
    echo "Generating templ files..." && \
    rm -rf **/*_templ.go && \
    sh -c "cd core && templ generate" && \
    echo "Generating sqlc queires..." && \
    sh -c "./scripts/sqlc-gen.sh ./core" && \
    echo "Preparing system plugins (statically links the Devkit theme into core/plugin.so)..." && \
    go run --tags="$GO_TAGS" ./core/cmd/sysplugin-prepare/main.go && \
    echo "Linking system plugin modules into core/go.mod (require)..." && \
    link_system_plugin_modules && \
    echo "Generating core/product.json (prebuilt stand-in; devkit ships no gen-product-version source)..." && \
    go run --tags="$GO_TAGS" ./core/cmd/gen-product-version/main.go && \
    echo "Compiling livereload binary (devkit ships no core/cmd source)..." && \
    go build --tags="$GO_TAGS" -o bin/livereload ./core/cmd/livereload && \
		go run --tags="$GO_TAGS" ./core/cmd/create-devkit/main.go
