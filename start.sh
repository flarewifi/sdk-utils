#!/bin/sh

export FLARE_DIR="/opt/flarehotspot"
export APP_DIR="$FLARE_DIR/app"
export APP_TMP="$FLARE_DIR/tmp"
export DATA_DIR="$FLARE_DIR/data"
export STORAGE_DIR="$DATA_DIR/storage"
export SOFTWARE_UPDATE_DIR="$STORAGE_DIR/system/updates"
export BACKUP_DIR="$STORAGE_DIR/system/backup"
export PATH="$APP_DIR/bin:$PATH"

apply_updates() {
    EXTRACT_TMP="$FLARE_DIR/extract_tmp" && \
    echo "\n\nFound software update, applying..." && \
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
        echo "\n\nOld version is available, reverting updates..." && \
            rm -rf $APP_DIR/* && \
            tar -xzf $BACKUP_DIR/backup.tar.gz -C $APP_DIR && \
            rm -rf $BACKUP_DIR && \
            cd $APP_DIR && \
            touch $APP_DIR/.reverted && \
            echo "Old version restored successfully."
    else
        echo "\n\nNo backup of old version is available, keeping current installation"
        return 1
    fi
}

link_data() {
    # Link data directory
    if [ ! -e "$DATA_DIR" ]; then
      echo "\n\nData directory $DATA_DIR does not exist, creating..." && \
      mkdir -p $DATA_DIR && \
      echo "\n\nData directory created at $DATA_DIR"
    fi

    if [ ! -e "$APP_DIR/data" ]; then
        (\
                echo "\n\nLinking data directory from $DATA_DIR to $APP_DIR/data" && \
                ln -s $DATA_DIR $APP_DIR/data && \
                echo "\n\nFiles in $APP_DIR/data: $(ls -l $APP_DIR/data/)"
            ) || ( \
                echo "\n\nFailed to link data directory, exiting" && \
                return 1
        )
    fi
}

start() {
    (\
            cd $APP_DIR && \
            echo "\n\nStarting Flare Hotspot from $APP_DIR" && \
            link_data && \
            mkdir -p $APP_TMP && \
            flare server
        ) || (\
            echo "\n\nFailed to start application, reverting to old version if available..." && \
            revert_updates
            $APP_DIR/start.sh
    )
}


if [ -e "$SOFTWARE_UPDATE_DIR" ] && ls $SOFTWARE_UPDATE_DIR/*.tar.gz 1> /dev/null 2>&1; then
    (apply_updates && $APP_DIR/start.sh) || (
        echo "\n\nFailed to apply updates!"
        revert_updates
        $APP_DIR/start.sh
    )
else
    start
fi
