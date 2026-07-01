#!/bin/sh

# Check for rsync and install it if missing.
# Works on both opkg (OpenWRT <= 24.10) and apk (OpenWRT >= 25.0 / snapshots).
#
# node/npm are intentionally NOT installed: frontend assets are bundled by the
# Go-native esbuild library against libraries vendored into each plugin's
# resources/assets, so nothing on-device runs `npm install`. (The apk feeds for
# the newest OpenWRT line no longer ship a binary `node` anyway.)

if command -v apk >/dev/null 2>&1; then
  PKG_UPDATE="apk update"
  PKG_INSTALL="apk add"
else
  PKG_UPDATE="opkg update"
  PKG_INSTALL="opkg install"
fi

# Check and install rsync
if ! command -v rsync >/dev/null 2>&1; then
  echo "rsync not found, installing..."
  $PKG_UPDATE && $PKG_INSTALL rsync
else
  echo "rsync is already installed."
fi
