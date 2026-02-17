package api

import (
	"core/utils/config"
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
	return NewAppCfgApi()
}

func (self *ConfigApi) Bandwidth() sdkapi.IBandwidthCfgApi {
	return NewBandwdCfgApi(self.api.SessionMgr)
}

func (self *ConfigApi) Plugin() sdkapi.IPluginCfgApi {
	return config.NewPluginCfgApi(self.api.info.Package)
}
