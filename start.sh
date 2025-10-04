#!/bin/sh

export APPDIR=$(dirname $(realpath $0))
export APPTMP="/var/tmp/flarehotspot"

export STORAGE_DIR="$APPDIR/data/storage"
export SOFTWARE_UPDATE_DIR="$STORAGE_DIR/system/update"
export BACKUP_DIR="$STORAGE_DIR/system/backup"

if [ -e "$SOFTWARE_UPDATE_DIR"]; then
  rm -rf $BACKUP_DIR
  mv $APPDIR $BACKUP_DIR
  mv $SOFTWARE_UPDATE_DIR $APPDIR
fi

mkdir -p $APPTMP && \
    ./bin/flare server
