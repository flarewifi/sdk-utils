package config

import (
	"fmt"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const databaseJsonFile = "database.json"

type DbConfig sdkutils.DbConfig

func (cfg *DbConfig) DbUrlString() string {
	return fmt.Sprintf("%s database=%s", cfg.BaseConnStr(), cfg.Database)
}

func (cfg *DbConfig) BaseConnStr() string {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s", cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.SslMode)

	return connStr
}

func ReadDatabaseConfig() (*DbConfig, error) {
	var cfg DbConfig
	err := readConfigFile(databaseJsonFile, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Host == "" {
		cfg.Host = "localhost"
	}

	if cfg.SslMode == "" {
		cfg.SslMode = "disable"
	}

	if cfg.Port == 0 {
		cfg.Port = 5432
	}

	if cfg.SqlitePath == "" {
		cfg.SqlitePath = filepath.Join(sdkutils.PathDataDir, "db/database.sqlite")
	}

	cfg.Database = strings.ToLower(cfg.Database)

	return &cfg, nil
}

func WriteDatabaseConfig(cfg DbConfig) error {
	return writeConfigFile(databaseJsonFile, cfg)
}
