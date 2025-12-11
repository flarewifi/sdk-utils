//go:build sqlite && !cgo

package db

import (
	_ "modernc.org/sqlite" // Pure-Go SQLite driver
)

const SqliteDriverName = "sqlite"
