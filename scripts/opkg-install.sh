#!/bin/sh

# Check for node, node-npm, and rsync and install them if missing

# Check and install node
if ! command -v node >/dev/null 2>&1; then
  echo "node not found, installing..."
  opkg update && opkg install node
else
  echo "node is already installed."
fi

# Check and install npm
if ! command -v npm >/dev/null 2>&1; then
  echo "npm not found, installing..."
  opkg update && opkg install node-npm
else
  echo "npm is already installed."
fi

# Check and install rsync
if ! command -v rsync >/dev/null 2>&1; then
  echo "rsync not found, installing..."
  opkg update && opkg install rsync
else
  echo "rsync is already installed."
fi
