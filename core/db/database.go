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
	mu         sync.Mutex
	db         *pgxpool.Pool
	readyCalls int
	Queries    queries.Queries
	ConnErr    error
}

func NewDatabase() *Database {

	var db Database

	// Run in separate routine to show the booting page earlier.
	go func(db *Database) {
		db.mu.Lock()
		defer db.mu.Unlock()

		dbpass := sdkutils.RandomStr(8)
		dbname := strings.ToLower(fmt.Sprintf("flarehotspot_%s", sdkutils.RandomStr(8)))

		// Setup PostgreSQL server
		err := pg.SetupServer(dbpass, dbname)
		if err != nil {
			log.Println("Error installing postgres db: ", err)
			db.ConnErr = err
			return
		}

		cfg, err := config.ReadDatabaseConfig()
		if err != nil {
			db.ConnErr = err
			return
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
				time.Sleep(time.Duration(portCheckIndex) * time.Second)
			}
		}

		if !portOK {
			db.ConnErr = fmt.Errorf("Unable to connect to the %s postgres server!", cfg.Host)
			return
		}

		ctx := context.Background()
		conn, err := pgx.Connect(ctx, cfg.BaseConnStr())
		if err != nil {
			log.Println("Error opening database: ", err)
			db.ConnErr = err
			return
		}
		defer conn.Close(ctx)

		err = pg.CreateDb(ctx, conn)
		if err != nil {
			db.ConnErr = err
			return
		}

		url := cfg.DbUrlString()
		log.Println("DB URL: ", url)

		dbConf, err := pgxpool.ParseConfig(url)
		if err != nil {
			db.ConnErr = err
			return
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
			db.ConnErr = err
			return
		}

		// TODO: find an equivalent postgresql sql query debugging

		err = CheckDatabaseConnection(pgPool)
		if err != nil {
			db.ConnErr = err
			return
		}

		db.Queries = *queries.New(pgPool)
		db.db = pgPool
	}(&db)

	return &db
}

func CheckDatabaseConnection(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return pool.Ping(ctx)
}

func (d *Database) WaitReady() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.readyCalls > 0 {
		panic("Database WaitReady() called more than once!")
	}
	d.readyCalls++
}

func (d *Database) SqlDB() (db *pgxpool.Pool) {
	if d.ConnErr != nil {
		log.Println("Unable to connect to database: ", d.ConnErr)
	}
	return d.db
}

func (d *Database) SetSql(db *pgxpool.Pool) {
	d.db = db
}
