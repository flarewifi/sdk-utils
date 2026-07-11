#!/bin/sh

# Non-mono boot + staged-update applier.
#
# Unlike start-mono.sh (which replaces the whole app from a single tarball),
# a non-mono install updates the core and each plugin as INDEPENDENT, already-built
# packages. Each staged package is a directory under $SOFTWARE_UPDATE_DIR named by
# its package id:
#
#   $SOFTWARE_UPDATE_DIR/com.flarego.core/      -> overlays the app/core layer
#   $SOFTWARE_UPDATE_DIR/com.flarego.<plugin>/  -> replaces plugins/installed/<plugin>
#
# A core update and its ABI-matched plugins are staged together and applied here
# atomically on the next boot (a Go plugin .so is ABI-locked to its core build).
# Apply is OVERLAY, not wipe: the data/ symlink and any plugins not part of the
# update are left untouched. The .staged_complete marker is written by the app
# only after the full set is staged, so a partial download is never applied.

export GOTOOLCHAIN=go1.21.13
export FLARE_DIR="/opt/flarewifi"
export APP_DIR="$FLARE_DIR/app"
export APP_TMP="$FLARE_DIR/tmp"
export DATA_DIR="$FLARE_DIR/data"
export STORAGE_DIR="$DATA_DIR/storage"
export SOFTWARE_UPDATE_DIR="$STORAGE_DIR/system/updates"
export BACKUP_DIR="$STORAGE_DIR/system/backup"
export PATH="$APP_DIR/bin:$PATH"

# Marker written by the init service's stop() so this script can tell an
# intentional shutdown from a crash (see start()). The default keeps the check
# working even when start.sh is run directly.
export STOP_MARKER="${STOP_MARKER:-$APP_TMP/.stopping}"

# Bounds the rollback -> re-exec boot loop so a persistently failing build can't
# spin forever. Carried across re-execs via the environment and reset to 0 on a
# real reboot (the init service starts this script with it unset). When the budget
# is exhausted, start() exits non-zero and the init service falls through to
# rescue mode.
MAX_BOOT_ATTEMPTS=3

STAGED_COMPLETE_MARKER="$SOFTWARE_UPDATE_DIR/.staged_complete"
CORE_PKG="com.flarego.core"

# dest_for_pkg prints the install destination for a staged package id.
dest_for_pkg() {
    if [ "$1" = "$CORE_PKG" ]; then
        echo "$APP_DIR"
    else
        echo "$APP_DIR/plugins/installed/$1"
    fi
}

# backup_pkg snapshots the current install of one package into $BACKUP_DIR/$pkg so
# a failed apply or a failed boot can be rolled back. For the core we snapshot only
# the top-level entries the staged payload will overwrite (never data/ or
# plugins/installed/). For a plugin we snapshot its whole install dir.
backup_pkg() {
    pkg="$1"
    staged="$SOFTWARE_UPDATE_DIR/$pkg"
    backup="$BACKUP_DIR/$pkg"
    rm -rf "$backup" && mkdir -p "$backup" || return 1

    if [ "$pkg" = "$CORE_PKG" ]; then
        for entry in "$staged"/*; do
            [ -e "$entry" ] || continue
            name="$(basename "$entry")"
            # data/ is owned by the persistent layer and is never overlaid by the core
            # apply (see apply_pkg), so never snapshot/restore the app/data symlink.
            [ "$name" = "data" ] && continue
            if [ -e "$APP_DIR/$name" ]; then
                cp -a "$APP_DIR/$name" "$backup/" || return 1
            fi
        done
    else
        dest="$(dest_for_pkg "$pkg")"
        if [ -e "$dest" ]; then
            cp -a "$dest" "$backup/payload" || return 1
        fi
    fi
}

# apply_pkg overlays one staged package onto its destination.
apply_pkg() {
    pkg="$1"
    staged="$SOFTWARE_UPDATE_DIR/$pkg"
    dest="$(dest_for_pkg "$pkg")"
    echo "Applying staged update: $pkg -> $dest"

    if [ "$pkg" = "$CORE_PKG" ]; then
        # app/data is a SYMLINK to the persistent $DATA_DIR (/opt/flarewifi/data),
        # which holds the device's LIVE state: data/config and data/db must survive the
        # update untouched. The release tarball bundles a data/ tree at its root, and the
        # staged core payload is that whole tarball — so a blanket `cp -a staged/. app/`
        # would drive the staged data/ onto the symlink and overwrite the live config +
        # database with the release's defaults (this is what reset config+sessions on the
        # first update).
        #
        # A core update must DISCARD the staged data/config and data/db entirely. The
        # ONLY part of the staged data/ that belongs in the persistent store is the
        # plugin SOURCES (data/plugins/local and data/plugins/devel), which the device
        # recompiles against the new core — copy just those into $DATA_DIR (writing to
        # the real dir, NOT through the app/data symlink), then drop the whole staged
        # data/ so the app overlay never touches it.
        if [ -d "$staged/data" ]; then
            for sub in plugins/local plugins/devel; do
                if [ -d "$staged/data/$sub" ]; then
                    mkdir -p "$DATA_DIR/$sub" || return 1
                    cp -a "$staged/data/$sub"/. "$DATA_DIR/$sub"/ || return 1
                fi
            done
            rm -rf "$staged/data"
        fi
        # defaults/ ships the release's fallback config templates (application.json,
        # plugins.json, etc. -- see TidyConfigFiles in go/builder/tasks/utils.go). Unlike
        # the rest of the core overlay, this must be a full REPLACE, not an additive
        # merge: a stale template from a previous release/device_config must not survive
        # just because the new release doesn't happen to ship a same-named file.
        rm -rf "$APP_DIR/defaults" || return 1

        # Overlay the remaining core entries onto the app dir. plugins/installed/ is
        # intentionally overlaid (ABI-matched plugins ship with the core); data/ is now
        # gone from the staged set, so the persistent symlink is left untouched.
        cp -a "$staged"/. "$APP_DIR"/ || return 1
    else
        # Replace the plugin's install dir wholesale with the staged build.
        mkdir -p "$APP_DIR/plugins/installed" || return 1
        rm -rf "$dest" && mkdir -p "$dest" && cp -a "$staged"/. "$dest"/ || return 1
    fi
}

# restore_all rolls every backed-up package back to its destination. Used when an
# apply step fails midway or the freshly-applied version fails to boot.
restore_all() {
    [ -d "$BACKUP_DIR" ] || return 0
    for b in "$BACKUP_DIR"/*; do
        [ -d "$b" ] || continue
        pkg="$(basename "$b")"
        dest="$(dest_for_pkg "$pkg")"
        echo "Rolling back: $pkg"
        if [ "$pkg" = "$CORE_PKG" ]; then
            cp -a "$b"/. "$APP_DIR"/ 2>/dev/null
        elif [ -e "$b/payload" ]; then
            rm -rf "$dest" && mkdir -p "$(dirname "$dest")" && cp -a "$b/payload" "$dest"
        else
            # Package was newly added by this update (no prior install): remove it.
            rm -rf "$dest"
        fi
    done
}

apply_updates() {
    echo "Found staged updates, applying..."

    # Phase 1: back up the current install of every staged package.
    rm -rf "$BACKUP_DIR" && mkdir -p "$BACKUP_DIR" || return 1
    for d in "$SOFTWARE_UPDATE_DIR"/*; do
        [ -d "$d" ] || continue
        pkg="$(basename "$d")"
        backup_pkg "$pkg" || { echo "ERROR: backup failed for $pkg"; restore_all; return 1; }
    done

    # Phase 2: overlay each staged package; roll back the whole set on any failure.
    for d in "$SOFTWARE_UPDATE_DIR"/*; do
        [ -d "$d" ] || continue
        pkg="$(basename "$d")"
        apply_pkg "$pkg" || { echo "ERROR: apply failed for $pkg"; restore_all; return 1; }
    done

    # Clear the staged set; keep $BACKUP_DIR until the new version boots OK so a
    # boot failure in start() can still roll back.
    rm -rf "$SOFTWARE_UPDATE_DIR"/* "$STAGED_COMPLETE_MARKER"
    touch "$APP_DIR/.updated"
    echo "Staged updates applied successfully."
}

link_data() {
    if [ ! -e "$DATA_DIR" ]; then
        mkdir -p "$DATA_DIR"
    fi
    if [ ! -e "$APP_DIR/data" ]; then
        ln -s "$DATA_DIR" "$APP_DIR/data" || {
            echo "Failed to link data directory, exiting"
            return 1
        }
    fi
}

start() {
    # Boot-attempt guard: bound the rollback -> re-exec chain so a persistently
    # failing build can't loop forever. When the budget is spent we exit non-zero
    # so the init service falls through to rescue mode.
    if [ "${FLARE_BOOT_ATTEMPT:-0}" -ge "$MAX_BOOT_ATTEMPTS" ]; then
        echo "Reached $MAX_BOOT_ATTEMPTS failed boot attempts; exiting so rescue mode can take over."
        exit 1
    fi
    FLARE_BOOT_ATTEMPT=$(( ${FLARE_BOOT_ATTEMPT:-0} + 1 ))
    export FLARE_BOOT_ATTEMPT

    cd "$APP_DIR" || exit 1
    echo "Starting Flarewifi from $APP_DIR (boot attempt $FLARE_BOOT_ATTEMPT/$MAX_BOOT_ATTEMPTS)"
    link_data || exit 1
    mkdir -p "$APP_TMP"

    flare server
    EXIT_CODE=$?

    # A deliberate stop (the init service's stop()) drops $STOP_MARKER and SIGTERMs
    # flare (exit 143). Treat an intentional stop or any clean exit as "done": do
    # not roll back and do not restart, so a normal stop/reboot stays stopped.
    if [ -e "$STOP_MARKER" ] || [ "$EXIT_CODE" -eq 0 ] || [ "$EXIT_CODE" -eq 143 ]; then
        echo "flare server exited (code $EXIT_CODE); intentional or clean shutdown, not restarting."
        # A clean exit (as opposed to the crash branch below) is the confirmation
        # that whatever update apply_updates staged this boot session is good --
        # nothing else in the codebase ever clears $BACKUP_DIR on success (it was
        # only ever removed on the CRASH path below, or on a failed apply). Left
        # uncleared, a stale backup from an already-confirmed-good update sits
        # there indefinitely; if flare later crashes for a reason UNRELATED to
        # that update (possibly boots/reboots later), the crash branch's
        # restore_all still finds it and silently reverts core/product.json (and
        # the rest of core/) back to the pre-update version -- which looks exactly
        # like the update "didn't take" even though it applied correctly.
        rm -rf "$BACKUP_DIR"
        exit 0
    fi

    echo "flare server crashed (code $EXIT_CODE), rolling back staged update if any..."
    restore_all
    rm -rf "$BACKUP_DIR"
    # exec replaces this shell so the re-run reads start.sh fresh from disk instead
    # of continuing to read a script file the overlay may have just rewritten.
    exec "$APP_DIR/start.sh"
}

if [ -e "$STAGED_COMPLETE_MARKER" ]; then
    # Adopt the STAGED core's start.sh BEFORE backup_pkg/apply_pkg/restore_all ever
    # run, so a fix to THIS apply/rollback logic takes effect for the very update
    # that ships it, instead of staying governed by the OLD (possibly buggy)
    # start.sh until the update after next. A plugin-only update has no staged
    # com.flarego.core, so staged_start_sh naturally doesn't exist and this is
    # skipped. FLARE_STARTSH_ADOPTED (exported before the re-exec) guards against
    # re-adopting forever once this boot has already done so.
    staged_start_sh="$SOFTWARE_UPDATE_DIR/$CORE_PKG/start.sh"
    if [ -e "$staged_start_sh" ] && [ -z "$FLARE_STARTSH_ADOPTED" ]; then
        echo "Adopting staged start.sh before applying update..."
        if cp -a "$staged_start_sh" "$APP_DIR/start.sh"; then
            export FLARE_STARTSH_ADOPTED=1
            exec "$APP_DIR/start.sh"
        fi
        echo "WARNING: failed to adopt staged start.sh; continuing with the current one"
    fi

    # Apply the staged set, then exec the (possibly just-updated) start.sh fresh.
    # apply_updates clears the staged set + marker on success, so the re-exec'd
    # start.sh boots normally instead of re-applying. $BACKUP_DIR is intentionally
    # kept so a boot failure in the re-exec'd start() can still roll back.
    if apply_updates; then
        exec "$APP_DIR/start.sh"
    else
        echo "Failed to apply staged updates!"
        restore_all
        # $STAGED_COMPLETE_MARKER is a dotfile (.staged_complete) directly under
        # $SOFTWARE_UPDATE_DIR -- the "/*" glob does NOT match it, so it must be
        # named explicitly here too (the success path above already does this).
        # Omitting it left the marker behind after a failed apply: the payload dirs
        # are gone (wiped by the glob) but the marker survives, so the NEXT boot
        # re-enters this branch, finds nothing to apply/back up, trivially
        # "succeeds", and silently discards the staged core update instead of
        # retrying it or surfacing the original failure.
        rm -rf "$BACKUP_DIR" "$SOFTWARE_UPDATE_DIR"/* "$STAGED_COMPLETE_MARKER"
        exec "$APP_DIR/start.sh"
    fi
else
    # No complete marker: discard any partial/aborted staging and boot normally.
    if [ -e "$SOFTWARE_UPDATE_DIR" ]; then
        rm -rf "$SOFTWARE_UPDATE_DIR"/* 2>/dev/null
    fi
    start
fi
