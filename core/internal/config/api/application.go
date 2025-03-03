package cfgapi

import (
	"core/internal/config"
	sdkapi "sdk/api"
)

func NewAppCfgApi() *AppCfgApi {
	return &AppCfgApi{}
}

type AppCfgApi struct{}

func (c *AppCfgApi) Get() (sdkapi.AppCfg, error) {
	cfg, err := config.ReadApplicationConfig()
	if err != nil {
		return sdkapi.AppCfg{}, err
	}

	return sdkapi.AppCfg{
		Lang:     cfg.Lang,
		Currency: cfg.Currency,
		Secret:   cfg.Secret,
	}, nil
}

func (c *AppCfgApi) Save(cfg sdkapi.AppCfg) error {
	data := config.AppConfig{
		Lang:     cfg.Lang,
		Currency: cfg.Currency,
		Secret:   cfg.Secret,
	}

	return config.WriteApplicationConfig(data)
}
