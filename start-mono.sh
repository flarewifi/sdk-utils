#!/bin/sh

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

# Bounds the revert -> re-exec boot loop so a persistently failing build can't
# spin forever. Carried across re-execs via the environment and reset to 0 on a
# real reboot. When exhausted, start() exits non-zero and the init service falls
# through to rescue mode.
MAX_BOOT_ATTEMPTS=3

apply_updates() {
    EXTRACT_TMP="$FLARE_DIR/extract_tmp" && \
    echo "Found software update, applying..." && \
        # Clean old app directory
        rm -rf $APP_DIR/* && \
        echo "Cleaned old application files" && \
        # Clean extract tmp directory to free up space
        rm -rf $EXTRACT_TMP && \
        mkdir -p $EXTRACT_TMP && \
        echo "Extracting software update to $EXTRACT_TMP..." && \
        tar -xzf $SOFTWARE_UPDATE_DIR/*.tar.gz -C $EXTRACT_TMP && \
        echo "Finding application root directory..." && \
        BIN_DIR=$(find $EXTRACT_TMP -type d -name "bin" | head -n 1) && \
        if [ -z "$BIN_DIR" ]; then \
            echo "ERROR: No bin directory found in update package!" && \
            return 1; \
        fi && \
        APP_ROOT=$(dirname $BIN_DIR) && \
        echo "Found application root at $APP_ROOT" && \
        echo "Moving application files to $APP_DIR..." && \
        mv $APP_ROOT/* $APP_DIR/ && \
        echo "Cleaning up temporary files..." && \
        rm -rf $EXTRACT_TMP && \
        rm -rf $SOFTWARE_UPDATE_DIR && \
        cd $APP_DIR && \
        touch $APP_DIR/.updated && \
        echo "Software updates applied successfully."
}

revert_updates() {
    if [ -e $BACKUP_DIR/backup.tar.gz ]; then
        echo "Old version is available, reverting updates..." && \
            rm -rf $APP_DIR/* && \
            tar -xzf $BACKUP_DIR/backup.tar.gz -C $APP_DIR && \
            rm -rf $BACKUP_DIR && \
            cd $APP_DIR && \
            touch $APP_DIR/.reverted && \
            echo "Old version restored successfully."
    else
        echo "No backup of old version is available, keeping current installation"
        return 1
    fi
}

link_data() {
    # Link data directory
    if [ ! -e "$DATA_DIR" ]; then
      echo "Data directory $DATA_DIR does not exist, creating..." && \
      mkdir -p $DATA_DIR && \
      echo "Data directory created at $DATA_DIR"
    fi

    if [ ! -e "$APP_DIR/data" ]; then
        (\
                echo "Linking data directory from $DATA_DIR to $APP_DIR/data" && \
                ln -s $DATA_DIR $APP_DIR/data && \
                echo "Files in $APP_DIR/data: $(ls -l $APP_DIR/data/)"
            ) || ( \
                echo "Failed to link data directory, exiting" && \
                return 1
        )
    fi
}

start() {
    # Boot-attempt guard: bound the revert -> re-exec chain so a persistently
    # failing build can't loop forever. When the budget is spent we exit non-zero
    # so the init service falls through to rescue mode.
    if [ "${FLARE_BOOT_ATTEMPT:-0}" -ge "$MAX_BOOT_ATTEMPTS" ]; then
        echo "Reached $MAX_BOOT_ATTEMPTS failed boot attempts; exiting so rescue mode can take over."
        exit 1
    fi
    FLARE_BOOT_ATTEMPT=$(( ${FLARE_BOOT_ATTEMPT:-0} + 1 ))
    export FLARE_BOOT_ATTEMPT

    cd $APP_DIR || exit 1
    echo "Starting Flarewifi from $APP_DIR (boot attempt $FLARE_BOOT_ATTEMPT/$MAX_BOOT_ATTEMPTS)"
    link_data || exit 1
    mkdir -p $APP_TMP

    flare server
    EXIT_CODE=$?

    # A deliberate stop (the init service's stop()) drops $STOP_MARKER and SIGTERMs
    # flare (exit 143). Treat an intentional stop or any clean exit as "done": do
    # not revert and do not restart, so a normal stop/reboot stays stopped.
    if [ -e "$STOP_MARKER" ] || [ "$EXIT_CODE" -eq 0 ] || [ "$EXIT_CODE" -eq 143 ]; then
        echo "flare server exited (code $EXIT_CODE); intentional or clean shutdown, not restarting."
        exit 0
    fi

    echo "flare server crashed (code $EXIT_CODE), reverting to old version if available..."
    revert_updates
    # exec replaces this shell so the re-run reads start.sh fresh from disk instead
    # of continuing to read a script file apply/revert may have just rewritten.
    exec "$APP_DIR/start.sh"
}


DOWNLOAD_COMPLETE_MARKER="$SOFTWARE_UPDATE_DIR/.dl_software_update_complete"

if [ -e "$SOFTWARE_UPDATE_DIR" ] && ls "$SOFTWARE_UPDATE_DIR"/*.tar.gz 1> /dev/null 2>&1; then
    if [ -e "$DOWNLOAD_COMPLETE_MARKER" ]; then
        # apply_updates clears $SOFTWARE_UPDATE_DIR on success, so the re-exec'd
        # start.sh boots normally instead of re-applying. exec replaces this shell
        # so we read the freshly-installed start.sh from offset 0.
        if apply_updates; then
            exec "$APP_DIR/start.sh"
        else
            echo "Failed to apply updates!"
            revert_updates
            # Drop the staged tarball so the re-exec doesn't try to apply it again.
            rm -rf "$SOFTWARE_UPDATE_DIR"/*.tar.gz "$DOWNLOAD_COMPLETE_MARKER"
            exec "$APP_DIR/start.sh"
        fi
    else
        echo "Update files found but download incomplete (marker missing), skipping update..."
        rm -rf "$SOFTWARE_UPDATE_DIR"/*.tar.gz
        start
    fi
else
    start
fi
