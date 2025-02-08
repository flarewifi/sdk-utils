package api

import (
	"core/internal/config"
	cfgapi "core/internal/config/api"
	sdkapi "sdk/api"
)

func NewConfigApi(api *PluginApi) {
	cfgApi := &ConfigApi{api}
	api.ConfigAPI = cfgApi
}

type ConfigApi struct {
	api *PluginApi
}

func (self *ConfigApi) Application() sdkapi.IAppCfgApi {
	return cfgapi.NewAppCfgApi()
}

func (self *ConfigApi) Bandwidth() sdkapi.IBandwidthCfgApi {
	return cfgapi.NewBandwdCfgApi()
}

func (self *ConfigApi) Plugin() sdkapi.IPluginCfgApi {
	return config.NewPluginCfgApi(self.api.info.Package)
}
