package config

import (
	"fmt"
	"path/filepath"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const applicationJsonFile = "application.json"

var SupportedLanguages = []string{
	"en", // English
	"af", // Afrikaans
}

var defaultAppCfg = sdkapi.AppConfig{
	Lang:     "en",
	Currency: "USD",
	Secret:   sdkutils.RandomStr(16),
	Channel:  "stable",
}

func ReadApplicationConfig() (sdkapi.AppConfig, error) {
	var cfg sdkapi.AppConfig

	err := readConfigFile(applicationJsonFile, &cfg)
	if err != nil {
		// generate defaults if not exists
		fmt.Println(err)
		fmt.Println("Generating default application configuration...")
		defaultFile := filepath.Join(sdkutils.PathDefaultsDir, applicationJsonFile)
		writeConfigFile(defaultFile, defaultAppCfg)
		return defaultAppCfg, err
	}

	if cfg.Lang == "" {
		cfg.Lang = defaultAppCfg.Lang
	}

	if cfg.Currency == "" {
		cfg.Currency = defaultAppCfg.Currency
	}

	if cfg.Secret == "" {
		cfg.Secret = defaultAppCfg.Secret
	}

	if cfg.Channel == "" {
		cfg.Channel = defaultAppCfg.Channel
	}

	return cfg, nil
}

func WriteApplicationConfig(cfg sdkapi.AppConfig) error {
	return writeConfigFile(applicationJsonFile, cfg)
}
