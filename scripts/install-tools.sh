#!/bin/sh

echo "Checking and installing CLI tools..."

# Check and install reflex
if ! command -v reflex >/dev/null 2>&1; then
  echo "reflex not found, installing..."
  go install github.com/cespare/reflex@v0.3.1
else
  echo "reflex is already installed."
fi

# Check and install sqlc
if ! command -v sqlc >/dev/null 2>&1; then
  echo "sqlc not found, installing..."
  go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0
else
  echo "sqlc is already installed."
fi

# Check and install templ
if ! command -v templ >/dev/null 2>&1; then
  echo "templ not found, installing..."
  go install github.com/a-h/templ/cmd/templ@v0.2.793
else
  echo "templ is already installed."
fi

