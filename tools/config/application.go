package config

import (
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const applicationJsonFile = "application.json"

func ReadApplicationConfig() (sdkapi.AppConfig, error) {
	var cfg sdkapi.AppConfig

	err := readConfigFile(applicationJsonFile, &cfg)
	if err != nil {
		// generate defaults if not exists
		cfg := sdkapi.AppConfig{
			Lang:     "en",
			Currency: "USD",
			Secret:   sdkutils.RandomStr(16),
			Channel:  "stable",
		}

		err = writeConfigFile(applicationJsonFile, cfg)
		if err != nil {
			return cfg, err
		}

		return cfg, nil
	}

	return cfg, nil
}

func WriteApplicationConfig(cfg sdkapi.AppConfig) error {
	return writeConfigFile(applicationJsonFile, cfg)
}
