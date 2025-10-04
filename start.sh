#!/bin/sh

export APPDIR="/etc/flarehotspot"
export APPTMP="/var/tmp/flarehotspot"
export DATA_DIR="/var/lib/flarehotspot"

export STORAGE_DIR="$DATA_DIR/storage"
export SOFTWARE_UPDATE_DIR="$STORAGE_DIR/system/update"
export BACKUP_DIR="$STORAGE_DIR/system/backup"

apply_updates() {
    echo "\n\nFound software update, copying..." && \
        rm -rf $BACKUP_DIR && \
        cp -r $APPDIR $BACKUP_DIR && \
        rm -rf $APPDIR/* && \
        cp -r $SOFTWARE_UPDATE_DIR/* $APPDIR && \
        rm -rf $SOFTWARE_UPDATE_DIR && \
        cd $APPDIR && \
        echo "Software updates copied successfully."
}

revert_updates() {
    if [ -e $BACKUP_DIR ]; then
        echo "\n\nOld version is available, reverting updates..." && \
            rm -rf $APPDIR/* && \
            cp -r $BACKUP_DIR/* $APPDIR && \
            rm -rf $BACKUP_DIR && \
            cd $APPDIR && \
            echo "Old version copied successfully."
    else
        echo "\n\nNo backup of old version is available" && exit 1
    fi
}

link_data() {
    # Link data directory
    if [ ! -e "./data" ]; then
        (\
                echo "\n\nLinking data directory from $DATA_DIR to $APPDIR/data" && \
                ln -s $DATA_DIR $APPDIR/data && \
                echo "\n\nFiles in $APPDIR/data: $(ls -l $APPDIR/data/)"
            ) || ( \
                echo "\n\nFailed to link data directory, exiting" && \
                return 1
        )
    fi
}

if [ -e "$SOFTWARE_UPDATE_DIR" ]; then
    apply_updates || (echo "\n\nFailed to apply updates!" && revert_updates)
fi

start() {
    (\
            cd $APPDIR && \
            echo "\n\nStarting Flare Hotspot from $APPDIR" && \
            link_data && \
            mkdir -p $APPTMP && \
            ./bin/flare server
        ) || (\
            echo "\n\nFailed to start application, reverting to old version if available..." && \
            revert_updates && start
    )
}

start
