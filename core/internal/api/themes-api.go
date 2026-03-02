package api

import (
	sdkapi "sdk/api"
)

func NewThemesApi(api *PluginApi) {
	t := &ThemesApi{api: api}
	api.ThemesAPI = t
}

type ThemesApi struct {
	api         *PluginApi
	AdminTheme  *sdkapi.AdminThemeOpts
	PortalTheme *sdkapi.PortalThemeOpts
}

func (self *ThemesApi) NewAdminTheme(theme sdkapi.AdminThemeOpts) {
	self.AdminTheme = &theme
}

func (self *ThemesApi) NewPortalTheme(theme sdkapi.PortalThemeOpts) {
	self.PortalTheme = &theme
}

func (self *ThemesApi) GetAdminTheme() sdkapi.IPluginApi {
	pluginApi, _, err := self.api.PluginsMgrApi.GetAdminTheme()
	if err != nil {
		return nil
	}
	return pluginApi
}

func (self *ThemesApi) GetPortalTheme() sdkapi.IPluginApi {
	pluginApi, _, err := self.api.PluginsMgrApi.GetPortalTheme()
	if err != nil {
		return nil
	}
	return pluginApi
}
