//go:build !sqlite

package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"time"

	queries "core/db/queries"
	"core/internal/utils/pg"
	"tools/config"

	sdkutils "github.com/flarehotspot/sdk-utils"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	Driver = DriverPostgres
)

// NewDatabase initializes either PostgreSQL or SQLite depending on config.
func NewDatabase() *Database {
	cfg, err := config.ReadDatabaseConfig()
	if err != nil {
		log.Println("Error reading DB config:", err)

		cfg, err = generateDbConfig()
		if err != nil {
			log.Println("Error generating DB config:", err)
			return &Database{ConnErr: err}
		}
	}

	return newPostgresDatabase(cfg)
}

func generateDbConfig() (*config.DbConfig, error) {
	fmt.Println("Generating new Postgres database configuration...")
	cfg := &config.DbConfig{
		Host:     "localhost",
		Port:     5432,
		Database: fmt.Sprintf("flarewifi_%s", sdkutils.RandomStr(8)),
		Username: "postgres",
		Password: sdkutils.RandomStr(12),
		SslMode:  "disable",
	}

	defaultFile := filepath.Join(sdkutils.PathDefaultsDir, "database.json")
	userFile := filepath.Join(sdkutils.PathConfigDir, "database.json")

	if err := sdkutils.JsonWrite(defaultFile, &cfg); err != nil {
		return nil, err
	}

	if err := sdkutils.JsonWrite(userFile, &cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func newPostgresDatabase(cfg *config.DbConfig) *Database {
	fmt.Println("Using Postgres database at", cfg.Host)

	var db Database

	go func(db *Database) {
		db.mu.Lock()
		defer db.mu.Unlock()

		dbpass := cfg.Password
		dbname := cfg.Database

		// Setup PostgreSQL server
		if err := pg.SetupServer(dbpass, dbname); err != nil {
			log.Println("Error installing postgres db:", err)
			db.ConnErr = err
			return
		}

		// Wait for PostgreSQL to start
		for i := 0; i < 30; i++ {
			if ok := pg.CheckPostgresPort(cfg.Host, cfg.Port); ok {
				if ok, err := pg.CheckDBReady(context.Background(), cfg.BaseConnStr()); ok && err == nil {
					goto CONNECT
				}
			}
			time.Sleep(time.Duration(i+1) * time.Second)
		}
		db.ConnErr = fmt.Errorf("Unable to connect to postgres on %s", cfg.Host)
		return

	CONNECT:
		conn, err := sql.Open("pgx", cfg.BaseConnStr())
		if err != nil {
			db.ConnErr = err
			return
		}
		defer conn.Close()

		if err := pg.CreateDb(context.Background(), conn); err != nil {
			db.ConnErr = err
			return
		}

		url := cfg.DbUrlString()
		log.Println("DB URL:", url)

		sqlDB, err := sql.Open("pgx", url)
		if err != nil {
			db.ConnErr = err
			return
		}

		sqlDB.SetConnMaxLifetime(4 * time.Minute)
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)

		for retries := 0; retries < 5; retries++ {
			if err = sqlDB.PingContext(context.Background()); err == nil {
				break
			}
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			db.ConnErr = err
			return
		}

		db.DB = sqlDB
		db.Queries = *queries.New(sqlDB)
	}(&db)

	return &db
}
