package config

import (
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const applicationJsonFile = "application.json"

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
		writeConfigFile(applicationJsonFile, defaultAppCfg)
		return defaultAppCfg, err
	}

	if cfg.Lang == "" {
		cfg.Lang = defaultAppCfg.Lang
	}

	if cfg.Currency == "" {
		cfg.Currency = defaultAppCfg.Currency
	}

	if cfg.Secret == "" {
		cfg.Secret = sdkutils.RandomStr(16)
	}

	if cfg.Channel == "" {
		cfg.Channel = defaultAppCfg.Channel
	}

	return cfg, nil
}

func WriteApplicationConfig(cfg sdkapi.AppConfig) error {
	return writeConfigFile(applicationJsonFile, cfg)
}
