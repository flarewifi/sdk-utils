//go:build sqlite && cgo

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

// ResetDatabase drops all tables and re-runs migrations
// For SQLite: deletes the database file and recreates it
// Returns the new database connection that should replace the old one
// pluginMigrationsFn is called after core migrations to run plugin migrations
func ResetDatabase(sqldb *sql.DB, pluginMigrationsFn func(*sql.DB) error) (*sql.DB, error) {
	log.Println("Starting database reset (SQLite with CGO driver)...")

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

	// Reconnect to the database (this will create a new empty database)
	log.Println("Reconnecting to database...")
	newDB, err := sql.Open(db.SqliteDriverName, dbCfg.SqlitePath)
	if err != nil {
		return nil, fmt.Errorf("failed to reconnect to database: %w", err)
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
