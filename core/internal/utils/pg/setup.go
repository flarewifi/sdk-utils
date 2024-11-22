package pg

import (
	"fmt"
	"os"
	"path/filepath"
	stdstr "strings"
	"time"

	"core/internal/utils/cmd"

	gouci "github.com/digineo/go-uci"
	fs "github.com/flarehotspot/go-utils/fs"
	paths "github.com/flarehotspot/go-utils/paths"
	"github.com/goccy/go-json"
)

var (
	configPath = filepath.Join(paths.ConfigDir, "database.json")
	srvPgDir   = "/srv/pg/"
)

// Sets up all necessary postgresql database requirements
func SetupDb(dbpass string, dbname string) error {
	if isInstalled() {
		return nil
	}

	if err := prepPgSrvDir(); err != nil {
		return err
	}

	if err := prepPgConf(); err != nil {
		return err
	}

	if err := prepPgSrvConf(); err != nil {
		rmPgSrvDir()
		return err
	}

	if err := installPg(); err != nil {
		rmPgSrvDir()
		stopDb()
		return err
	}

	if err := setRootPass(dbpass); err != nil {
		return err
	}

	if err := createDb(dbname); err != nil {
		return err
	}

	if err := writeConfig(dbpass, dbname); err != nil {
		rmPgSrvDir()
		return err
	}

	return nil
}

func isInstalled() bool {
	return fs.Exists(srvPgDir) && fs.Exists(configPath)
}

func prepPgConf() error {
	pgConfPath := "/var/lib/postgresql/data/pgdata/postgresql.conf"
	bytes, err := os.ReadFile(pgConfPath)
	if err != nil {
		return err
	}

	content := string(bytes)
	if stdstr.Contains(content, "data_directory") {
		return nil
	}

	content += "\n"
	content += fmt.Sprintf("data_directory = '%s'\n", srvPgDir)
	content += "log_directory = '/var/log/postgresql'\n"
	content += "log_filename = 'postgresql.log'\n"

	return os.WriteFile(pgConfPath, []byte(content), 0644)
}

func prepPgSrvDir() error {
	commands := []string{
		"mkdir -p " + srvPgDir,
		"chown -R postgres:postgres " + srvPgDir,
	}

	for _, c := range commands {
		err := cmd.Exec(c, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func prepPgSrvConf() error {
	values, ok := gouci.Get("postgresql", "general", "enabled")
	enabled := ok && len(values) > 0 && values[0] == "1"
	if !enabled {
		gouci.Set("postgresql", "general", "enabled", "1")
		return gouci.Commit()
	}
	return nil
}

func installPg() error {
	commands := []string{
		"pg_ctl initdb -D" + srvPgDir,
		"chown -R postgres:postgres " + srvPgDir,
		"service postgresql start",
		"service postgresql enable",
	}

	for _, c := range commands {
		err := cmd.Exec(c, nil)
		if err != nil {
			return err
		}
	}

	// allowance time for postgres to boot first
	// sleep 3s
	time.Sleep(3 * time.Second)

	return nil

}

func rmPgSrvDir() {
	cmd.Exec("rm -rf "+srvPgDir, nil)
}

func stopDb() {
	cmd.Exec("service postgresql stop", nil)
}

func setRootPass(dbpass string) error {
	command := fmt.Sprintf("postgres psql -c \"ALTER USER postgres WITH PASSWORD '%s';\"", dbpass)
	return cmd.Exec(command, nil)
}

func createDb(dbname string) error {
	command := fmt.Sprintf("postgres createdb %s ", dbname)
	return cmd.Exec(command, nil)
}

func writeConfig(dbpass string, dbname string) error {
	cfg := map[string]string{
		"host":     "localhost",
		"username": "postgres",
		"password": dbpass,
		"database": dbname,
	}

	bytes, err := json.Marshal(&cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, bytes, 6004)
}
