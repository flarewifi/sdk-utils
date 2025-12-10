package db

import (
	"context"
	queries "core/db/queries"
	"database/sql"
	"sync"
)

const (
	DriverPostgres = "postgres"
	DriverSqlite   = "sqlite"
)

type Database struct {
	mu         sync.Mutex
	readyCalls int
	DB         *sql.DB
	Queries    queries.Queries
	ConnErr    error
}

func (db *Database) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, opts)
}

func (db *Database) Close() error {
	return db.Close()
}

// ReopenConnection safely replaces the database connection
// This is used after database reset operations
func (db *Database) ReopenConnection(newDB *sql.DB) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Close old connection if it exists and is not already closed
	if db.DB != nil {
		_ = db.DB.Close() // Ignore error as connection may already be closed
	}

	// Update to new connection
	db.DB = newDB
	db.Queries = *queries.New(newDB)
}

func (db *Database) WaitReady() {
	if db.readyCalls > 0 {
		panic("Database WaitReady() called more than once!")
	}
	db.readyCalls++

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.DB == nil {
		panic("Database failed to initialize properly!")
	}
}
