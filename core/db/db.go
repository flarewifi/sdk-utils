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
