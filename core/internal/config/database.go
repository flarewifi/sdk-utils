package config

import (
	"fmt"
)

const databaseJsonFile = "database.json"

type DbConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SslMode  string `json:"sslmode"`
}

func (cfg *DbConfig) DbUrlString() string {
	return fmt.Sprintf("%s/%s?sslmode=disable", cfg.BaseConnStr(), cfg.Database)
}

func (cfg *DbConfig) BaseConnStr() string {
	var password string
	if cfg.Password != "" {
		password = ":" + cfg.Password
	} else {
		password = ""
	}

	var port string
	if cfg.Port != 0 {
		port = fmt.Sprintf(":%d", cfg.Port)
	} else {
		port = ""
	}

	return fmt.Sprintf("postgres://%s%s@%s%s", cfg.Username, password, cfg.Host, port)
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

	return &cfg, nil
}

func WriteDatabaseConfig(cfg DbConfig) error {
	return writeConfigFile(databaseJsonFile, cfg)
}
