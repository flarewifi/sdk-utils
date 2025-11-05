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
