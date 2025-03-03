package db

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	queries "core/db/queries"
	"core/internal/config"
	"core/internal/utils/pg"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	mu      sync.RWMutex
	db      *pgxpool.Pool
	Queries queries.Queries
}

func NewDatabase() (*Database, error) {
	dbpass := sdkutils.RandomStr(8)
	dbname := strings.ToLower(fmt.Sprintf("flarehotspot_%s", sdkutils.RandomStr(8)))

	// Setup PostgreSQL server
	err := pg.SetupServer(dbpass, dbname)
	if err != nil {
		log.Println("Error installing postgres db: ", err)
		return nil, err
	}

	cfg, err := config.ReadDatabaseConfig()
	if err != nil {
		return nil, err
	}

	// Wait for the postgres server to be ready
	maxPortCheckTries := 30
	portCheckIndex := 0
	portOK := false
	for portCheckIndex < maxPortCheckTries {
		fmt.Println("Checking if database is up...")

		ok := pg.CheckPostgresPort(cfg.Host, cfg.Port)
		if ok {
			ok, err := pg.CheckDBReady(context.Background(), cfg.BaseConnStr())
			if ok && err == nil {
				portOK = true
				break
			}
		} else {
			portCheckIndex++
			time.Sleep(1 * time.Second)
		}
	}

	if !portOK {
		return nil, fmt.Errorf("Unable to connect to the %s postgres server!", cfg.Host)
	}

	var db Database

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.BaseConnStr())
	if err != nil {
		log.Println("Error opening database: ", err)
		return nil, err
	}
	defer conn.Close(ctx)

	err = pg.CreateDb(ctx, conn)
	if err != nil {
		return nil, err
	}

	url := cfg.DbUrlString()
	log.Println("DB URL: ", url)

	dbConf, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}

	//https://stackoverflow.com/questions/39980902/golang-mysql-error-packets-go33-unexpected-eof
	dbConf.MaxConnLifetime = time.Minute * 4

	// Ensure postgresql starts up during boot before returning err
	openErrorCountThreshold := 5
	pgPool, err := pgxpool.NewWithConfig(context.Background(), dbConf)
	for openErrorCount := 0; err != nil && openErrorCount < openErrorCountThreshold; openErrorCount++ {
		log.Println("Checking database connection...")
		pgPool, err = pgxpool.New(context.Background(), url)
		time.Sleep(time.Second * 2)
		log.Println("Error opening database: ", err)
	}
	if err != nil {
		log.Println("Error connecting to database.")
		return nil, err
	}

	// TODO: find an equivalent postgresql sql query debugging

	err = CheckDatabaseConnection(pgPool)
	if err != nil {
		return nil, err
	}

	db.Queries = *queries.New(pgPool)
	db.db = pgPool
	return &db, nil
}

func CheckDatabaseConnection(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return pool.Ping(ctx)
}

func (d *Database) SqlDB() (db *pgxpool.Pool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

func (d *Database) SetSql(db *pgxpool.Pool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.db = db
}
