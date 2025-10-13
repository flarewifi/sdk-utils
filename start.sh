#!/bin/sh

export FLARE_DIR="/opt/flarehotspot"
export APP_DIR="$FLARE_DIR/app"
export APP_TMP="$FLARE_DIR/tmp"
export DATA_DIR="$FLARE_DIR/data"
export STORAGE_DIR="$DATA_DIR/storage"
export SOFTWARE_UPDATE_DIR="$STORAGE_DIR/system/update"
export BACKUP_DIR="$STORAGE_DIR/system/backup"
export PATH="$APP_DIR/bin:$PATH"

apply_updates() {
    echo "\n\nFound software update, copying..." && \
        rm -rf $BACKUP_DIR && \
        cp -r $APP_DIR $BACKUP_DIR && \
        rm -rf $APP_DIR/* && \
        cp -r $SOFTWARE_UPDATE_DIR/* $APP_DIR && \
        rm -rf $SOFTWARE_UPDATE_DIR && \
        cd $APP_DIR && \
        touch $APP_DIR/.updated && \
        echo "Software updates copied successfully."
}

revert_updates() {
    if [ -e $BACKUP_DIR ]; then
        echo "\n\nOld version is available, reverting updates..." && \
            rm -rf $APP_DIR/* && \
            cp -r $BACKUP_DIR/* $APP_DIR && \
            rm -rf $BACKUP_DIR && \
            cd $APP_DIR && \
            touch $APP_DIR/.reverted && \
            echo "Old version copied successfully."
    else
        echo "\n\nNo backup of old version is available" && exit 1
    fi
}

link_data() {
    # Link data directory
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
            revert_updates && $APP_DIR/start.sh
    )
}


if [ -e "$SOFTWARE_UPDATE_DIR" ]; then
    (apply_updates && $APP_DIR/start.sh) || (echo "\n\nFailed to apply updates!" && \
        revert_updates && $APP_DIR/start.sh)
else
    start
fi
