//go:build postgres

package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"tools/config"
	"tools/migrate"
)

// ResetDatabase drops the database and recreates it with migrations
// For PostgreSQL: drops the database, recreates it, and runs migrations
// Returns the new database connection that should replace the old one
// pluginMigrationsFn is called after core migrations to run plugin migrations
func ResetDatabase(db *sql.DB, pluginMigrationsFn func(*sql.DB) error) (*sql.DB, error) {
	log.Println("Starting database reset (PostgreSQL)...")

	// Get database configuration
	dbCfg, err := config.ReadDatabaseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read database config: %w", err)
	}

	// Close the current database connection
	log.Println("Closing database connection...")
	if err := db.Close(); err != nil {
		return nil, fmt.Errorf("failed to close database: %w", err)
	}

	// Connect to postgres database (not the app database)
	log.Println("Connecting to postgres database...")
	postgresConnStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		dbCfg.Host, dbCfg.Port, dbCfg.Username, dbCfg.Password, dbCfg.SslMode)

	postgresDB, err := sql.Open("pgx", postgresConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer postgresDB.Close()

	// Terminate all connections to the target database
	log.Printf("Terminating connections to database: %s\n", dbCfg.Database)
	terminateQuery := fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()
	`, dbCfg.Database)

	if _, err := postgresDB.Exec(terminateQuery); err != nil {
		log.Printf("Warning: failed to terminate connections: %v\n", err)
	}

	// Drop the database
	log.Printf("Dropping database: %s\n", dbCfg.Database)
	dropQuery := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbCfg.Database)
	if _, err := postgresDB.Exec(dropQuery); err != nil {
		return nil, fmt.Errorf("failed to drop database: %w", err)
	}

	log.Println("Database dropped successfully")

	// Create the database
	log.Printf("Creating database: %s\n", dbCfg.Database)
	createQuery := fmt.Sprintf("CREATE DATABASE %s", dbCfg.Database)
	if _, err := postgresDB.Exec(createQuery); err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	log.Println("Database created successfully")

	// Close connection to postgres database
	postgresDB.Close()

	// Reconnect to the new database
	log.Println("Reconnecting to application database...")
	appConnStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbCfg.Host, dbCfg.Port, dbCfg.Username, dbCfg.Password, dbCfg.Database, dbCfg.SslMode)

	newDB, err := sql.Open("pgx", appConnStr)
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
