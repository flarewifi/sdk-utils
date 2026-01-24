//go:build cgo

package db

import (
	_ "github.com/mattn/go-sqlite3" // CGO SQLite driver for cross-compilation
)

const SqliteDriverName = "sqlite3"
