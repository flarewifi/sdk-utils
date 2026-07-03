package config

import (
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

const databaseJsonFile = "database.json"

type DbConfig sdkutils.DbConfig

var DefaultDbConfig = DbConfig{SqlitePath: filepath.Join(sdkutils.PathDataDir, "db/database.sqlite")}

func ReadDatabaseConfig() (*DbConfig, error) {
	var cfg DbConfig
	err := readConfigFile(databaseJsonFile, &cfg)
	if err != nil {
		return &DefaultDbConfig, nil
	}

	if cfg.SqlitePath == "" {
		return &DefaultDbConfig, nil
	}

	return &cfg, nil
}

func WriteDatabaseConfig(cfg DbConfig) error {
	return writeConfigFile(databaseJsonFile, cfg)
}
