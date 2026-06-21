//go:build cgo

package database

import (
	"database/sql"
	"fmt"
	"os"

	"core/db"
	"core/utils/config"
	"core/utils/migrate"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// sqliteFullDSN builds the SQLite connection URI with all required pragmas.
// Must match the DSN used in newSQLiteDatabase so the post-reset connection
// has the same WAL mode, busy timeout, and pool settings as the original.
func sqliteFullDSN(path string) string {
	return fmt.Sprintf("file:%s?_busy_timeout=10000&_journal_mode=WAL&_sync=NORMAL&_loc=UTC", path)
}

// ResetDatabase drops all tables and re-runs migrations
// For SQLite: deletes the database file and recreates it
// Returns the new database connection that should replace the old one
// pluginMigrationsFn is called after core migrations to run plugin migrations
func ResetDatabase(sqldb *sql.DB, pluginMigrationsFn func(*sql.DB) error) (*sql.DB, error) {
	dbCfg, err := config.ReadDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read database config: %w", err)
	}

	if err := sqldb.Close(); err != nil {
		return nil, fmt.Errorf("failed to close database: %w", err)
	}

	if err := os.Remove(dbCfg.SqlitePath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to delete database file: %w", err)
	}

	newDB, err := sql.Open(db.SqliteDriverName, sqliteFullDSN(dbCfg.SqlitePath))
	if err != nil {
		return nil, fmt.Errorf("failed to reconnect to database: %w", err)
	}

	newDB.SetConnMaxLifetime(0)
	newDB.SetMaxOpenConns(5)
	newDB.SetMaxIdleConns(2)

	newDB.Exec("PRAGMA wal_autocheckpoint = 1000")

	if err := migrate.Init(newDB); err != nil {
		newDB.Close()
		return nil, fmt.Errorf("failed to initialize migrations table: %w", err)
	}

	coreDir := sdkutils.PathCoreDir
	if err := migrate.MigrateUp(newDB, coreDir); err != nil {
		newDB.Close()
		return nil, fmt.Errorf("failed to run up migrations: %w", err)
	}

	if pluginMigrationsFn != nil {
		if err := pluginMigrationsFn(newDB); err != nil {
			newDB.Close()
			return nil, fmt.Errorf("failed to run plugin migrations: %w", err)
		}
	}

	return newDB, nil
}
