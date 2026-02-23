//go:build !cgo

package db

import (
	"context"
	"core/utils/config"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	queries "core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	Driver = DriverSqlite
)

func NewDatabase() *Database {
	var dbpath string
	dbcfg, err := config.ReadDatabaseConfig()
	if err != nil {
		dbpath = filepath.Join(sdkutils.PathDataDir, "db/database.sqlite")
	}
	dbpath = dbcfg.SqlitePath
	return newSQLiteDatabase(dbpath)
}

func newSQLiteDatabase(dbpath string) *Database {
	log.Println("Initializing SQLite database...")

	var db Database

	go func(db *Database) {
		db.mu.Lock()
		defer db.mu.Unlock()

		if !sdkutils.FsExists(dbpath) {
			if err := sdkutils.FsEnsureDir(filepath.Dir(dbpath)); err != nil {
				panic("Failed to create directories for SQLite DB: " + err.Error())
			}
			if _, err := os.Create(dbpath); err != nil {
				panic("Failed to create SQLite DB file: " + err.Error())
			}
		}

		dburl := fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL&_loc=UTC", dbpath)
		sqlDB, err := sql.Open(SqliteDriverName, dburl)
		if err != nil {
			log.Println("Error opening SQLite DB:", err)
			db.ConnErr = err
			return
		}

		sqlDB.SetConnMaxLifetime(0)
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)

		for retries := 0; retries < 5; retries++ {
			if err = sqlDB.PingContext(context.Background()); err == nil {
				break
			}
			time.Sleep(time.Second)
		}
		if err != nil {
			db.ConnErr = err
			return
		}

		// Force UTC timezone for all timestamp operations
		_, err = sqlDB.ExecContext(context.Background(), "PRAGMA timezone = 'UTC'")
		if err != nil {
			log.Println("Warning: Failed to set SQLite timezone to UTC:", err)
			// Don't fail - _loc=UTC in connection string should handle it
		}

		db.DB = sqlDB
		db.Queries = *queries.New(sqlDB)
	}(&db)

	return &db
}
