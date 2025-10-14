//go:build !dev

package pg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"core/internal/config"
	"core/internal/utils/cmd"

	gouci "github.com/digineo/go-uci"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	pgDataDir  = "/srv/postgresql/data"
	pgLogFile  = "/srv/postgresql/data/postgresql.log"
	pgPassFile = filepath.Join(sdkutils.PathTmpDir, "pg-pass.txt")
)

// Sets up all necessary postgresql server requirements
func SetupServer(dbpass string, dbname string) error {
	// Check if postgres is already setup
	var hasDbConfig bool
	_, err := config.ReadDatabaseConfig()
	if err == nil {
		hasDbConfig = true
	}

	isInstalled := sdkutils.FsExists(pgDataDir) && hasDbConfig
	if isInstalled {
		fmt.Println("Postgres is already setup.")
		return nil
	}

	// Prepare pg data directory
	if err := sdkutils.FsEnsureDir(pgDataDir); err != nil {
		return err
	}

	if err := cmd.Exec("chown -R postgres:postgres "+pgDataDir, nil); err != nil {
		return err
	}

	// Configure postgres service
	if ok := gouci.Set("postgresql", "config", "PGDATA", pgDataDir); !ok {
		return errors.New("uci: unable to set postgresql config PGDATA value")
	}

	if ok := gouci.Set("postgresql", "config", "PGLOG", pgLogFile); !ok {
		return errors.New("uci: unable to set postgresql config PGLOG value")
	}

	if err := gouci.Commit(); err != nil {
		fmt.Println("unable to commit postgresql config")
		return err
	}

	if err := os.WriteFile(pgPassFile, []byte(dbpass), sdkutils.PermFile); err != nil {
		fmt.Println("unable to write pg password file")
		return err
	}
	// don't forget to remove password file
	defer os.Remove(pgPassFile)

	initDbCmd := fmt.Sprintf(`sudo -u postgres sh -c "LC_COLLATE='C' initdb --pwfile=%s -D %s"`, pgPassFile, pgDataDir)
	if err := cmd.Exec(initDbCmd, &cmd.ExecOpts{
		Stdout: os.Stdout,
	}); err != nil {
		return err
	}

	// Write config file
	if err := config.WriteDatabaseConfig(config.DbConfig{
		Host:     "localhost",
		Username: "postgres",
		Password: dbpass,
		Database: dbname,
	}); err != nil {
		return err
	}

	// Enable postgresql service
	if err := cmd.Exec("service postgresql enable", nil); err != nil {
		fmt.Println("unable to enable postgresql service")
		return err
	}

	if err := cmd.Exec("service postgresql start", nil); err != nil {
		fmt.Println("unable to start postgresql service")
		return err
	}

	return nil
}
