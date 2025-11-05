package cfgapi

import (
	sdkapi "sdk/api"
	"tools/config"
)

func NewAppCfgApi() *AppCfgApi {
	return &AppCfgApi{}
}

type AppCfgApi struct{}

func (c *AppCfgApi) Get() (sdkapi.AppConfig, error) {
	cfg, err := config.ReadApplicationConfig()
	if err != nil {
		return sdkapi.AppConfig{}, err
	}

	return sdkapi.AppConfig{
		Lang:     cfg.Lang,
		Currency: cfg.Currency,
		Secret:   cfg.Secret,
		Channel:  cfg.Channel,
	}, nil
}

func (c *AppCfgApi) Save(cfg sdkapi.AppConfig) error {
	data := sdkapi.AppConfig{
		Lang:     cfg.Lang,
		Currency: cfg.Currency,
		Secret:   cfg.Secret,
		Channel:  cfg.Channel,
	}

	return config.WriteApplicationConfig(data)
}
