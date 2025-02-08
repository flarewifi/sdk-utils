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

func (self *ThemesApi) GetAdminAssets() (jsSrc string, cssHref string) {
	manifest := self.api.AssetsManifest

	if self.AdminTheme != nil {
		scriptFile, ok := manifest.AdminAssets.Scripts[self.AdminTheme.JsFile]
		if ok {
			jsSrc = self.api.HttpAPI.Helpers().AssetPath(scriptFile)
		}

		cssFile, ok := manifest.AdminAssets.Styles[self.AdminTheme.CssFile]
		if ok {
			cssHref = self.api.HttpAPI.Helpers().AssetPath(cssFile)
		}
	}

	return
}

func (self *ThemesApi) GetPortalAssets() (jsSrc string, cssHref string) {
	manifest := self.api.AssetsManifest

	if self.PortalTheme != nil {
		scriptFile, ok := manifest.PortalAssets.Scripts[self.PortalTheme.JsFile]
		if ok {
			jsSrc = self.api.HttpAPI.Helpers().AssetPath(scriptFile)
		}

		cssFile, ok := manifest.PortalAssets.Styles[self.PortalTheme.CssFile]
		if ok {
			cssHref = self.api.HttpAPI.Helpers().AssetPath(cssFile)
		}
	}

	return
}
