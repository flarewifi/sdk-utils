//go:build !cgo

package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"core/db"
	"core/utils/config"
	"core/utils/migrate"

	sdkutils "github.com/flarehotspot/sdk-utils"
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
	log.Println("Starting database reset (SQLite)...")

	// Get database configuration
	dbCfg, err := config.ReadDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read database config: %w", err)
	}

	// Close the database connection
	log.Println("Closing database connection...")
	if err := sqldb.Close(); err != nil {
		return nil, fmt.Errorf("failed to close database: %w", err)
	}

	// Delete the SQLite database file
	log.Printf("Deleting database file: %s\n", dbCfg.SqlitePath)
	if err := os.Remove(dbCfg.SqlitePath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to delete database file: %w", err)
	}

	log.Println("Database file deleted successfully")

	// Reconnect to the database (this will create a new empty database).
	// Use the full URI DSN with WAL mode and busy timeout so the post-reset
	// connection has the same tuning as the original (prevents 0 ms busy
	// timeout and DELETE journal mode on the freshly-opened connection).
	log.Println("Reconnecting to database...")
	newDB, err := sql.Open(db.SqliteDriverName, sqliteFullDSN(dbCfg.SqlitePath))
	if err != nil {
		return nil, fmt.Errorf("failed to reconnect to database: %w", err)
	}

	// Mirror the connection-pool settings from newSQLiteDatabase.
	newDB.SetConnMaxLifetime(0)
	newDB.SetMaxOpenConns(5)
	newDB.SetMaxIdleConns(2)

	// Re-apply WAL autocheckpoint (PRAGMA is per-connection in some drivers).
	if _, pragmaErr := newDB.Exec("PRAGMA wal_autocheckpoint = 1000"); pragmaErr != nil {
		log.Println("Warning: Failed to set WAL autocheckpoint after reset:", pragmaErr)
	}

	// Initialize migrations table
	log.Println("Initializing migrations table...")
	if err := migrate.Init(newDB); err != nil {
		newDB.Close()
		return nil, fmt.Errorf("failed to initialize migrations table: %w", err)
	}

	// Run up migrations to recreate schema
	log.Println("Running up migrations to recreate schema...")
	coreDir := sdkutils.PathCoreDir
	if err := migrate.MigrateUp(newDB, coreDir); err != nil {
		newDB.Close()
		return nil, fmt.Errorf("failed to run up migrations: %w", err)
	}

	log.Println("Database schema recreated successfully")

	// Run plugin migrations if callback provided
	if pluginMigrationsFn != nil {
		log.Println("Running plugin migrations...")
		if err := pluginMigrationsFn(newDB); err != nil {
			newDB.Close()
			return nil, fmt.Errorf("failed to run plugin migrations: %w", err)
		}
		log.Println("Plugin migrations completed successfully")
	}

	log.Println("Database reset completed successfully")

	return newDB, nil
}
