package config

import (
	sdkutils "github.com/flarehotspot/sdk-utils"
)

const applicationJsonFile = "application.json"

type AppConfig struct {
	Lang     string `json:"lang"`
	Currency string `json:"currency"`
	Secret   string `json:"secret"`
	Channel  string `json:"channel"`
}

func ReadApplicationConfig() (AppConfig, error) {
	var cfg AppConfig

	err := readConfigFile(applicationJsonFile, &cfg)
	if err != nil {
		// generate defaults if not exists
		cfg := AppConfig{
			Lang:     "en",
			Currency: "USD",
			Secret:   sdkutils.RandomStr(16),
		}

		err = writeConfigFile(applicationJsonFile, cfg)
		if err != nil {
			return cfg, err
		}

		return cfg, nil
	}

	return cfg, nil
}

func WriteApplicationConfig(cfg AppConfig) error {
	return writeConfigFile(applicationJsonFile, cfg)
}
