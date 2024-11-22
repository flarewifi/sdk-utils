package plugins

import (
	sdkhttp "sdk/api/http"
)

func NewThemesApi(api *PluginApi) {
	t := &HttpThemesApi{api: api}
	api.ThemesAPI = t
}

type HttpThemesApi struct {
	api         *PluginApi
	AdminTheme  *sdkhttp.AdminThemeOpts
	PortalTheme *sdkhttp.PortalThemeOpts
}

func (self *HttpThemesApi) NewAdminTheme(theme sdkhttp.AdminThemeOpts) {
	self.AdminTheme = &theme
}

func (self *HttpThemesApi) NewPortalTheme(theme sdkhttp.PortalThemeOpts) {
	self.PortalTheme = &theme
}

func (self *HttpThemesApi) GetAdminAssets() (jsSrc string, cssHref string) {
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

func (self *HttpThemesApi) GetPortalAssets() (jsSrc string, cssHref string) {
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
