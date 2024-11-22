package plugins

import (
	cfgapi "core/internal/config/api"
	sdkcfg "sdk/api/config"
)

func NewConfigApi(api *PluginApi) {
	cfgApi := &ConfigApi{api}
	api.ConfigAPI = cfgApi
}

type ConfigApi struct {
	api *PluginApi
}

func (self *ConfigApi) Application() sdkcfg.IAppCfgApi {
	return cfgapi.NewAppCfgApi()
}

func (self *ConfigApi) Bandwidth(ifname string) sdkcfg.IBandwidthCfgApi {
	return cfgapi.NewBandwdCfgApi(ifname)
}
