//go:build !sqlite

package pg

import (
	"context"
	"core/utils/config"
	"database/sql"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func CheckPostgresPort(host string, port int) bool {
	timeout := 2 * time.Second // Adjust timeout as needed

	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false // Port is not open
	}
	defer conn.Close()

	return true // Port is open
}

// CheckDBReady checks if the database server is ready to accept connections.
func CheckDBReady(ctx context.Context, connString string) (bool, error) {
	// Create a connection to the database
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return false, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// Test readiness by running a simple query
	err = conn.Ping(ctx)
	if err != nil {
		return false, fmt.Errorf("database is not ready: %w", err)
	}

	return true, nil
}

func CreateDb(ctx context.Context, conn *sql.DB) (err error) {
	cfg, err := config.ReadDatabaseConfig()
	if err != nil {
		return
	}

	log.Println("Creating database " + cfg.Database + "...")
	_, err = conn.Exec("CREATE DATABASE " + cfg.Database)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			log.Println("Database already exists, skipping creation.")
			return nil
		}

		return err
	}

	return nil
}
