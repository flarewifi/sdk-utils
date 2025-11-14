package api

import (
	"path/filepath"

	sdkapi "sdk/api"

	"core/resources/views/themes"
)

func NewPluginUtils(api *PluginApi) *PluginUtils {
	return &PluginUtils{api}
}

type PluginUtils struct {
	api *PluginApi
}

func (self *PluginUtils) Translate(msgtype string, msgk string, pairs ...any) string {
	translationsDir := self.Resource("translations")
	return TranslateMessage(translationsDir, msgtype, msgk, pairs...)
}

func (self *PluginUtils) Resource(path string) string {
	return filepath.Join(self.api.dir, "resources", path)
}

func GetAdminAssetsForPage(coreAPI *PluginApi, themeAPI *PluginApi, pluginAPI *PluginApi, v sdkapi.ViewPage, ga *GlobalAssets) (assets themes.AdminAssets, err error) {
	globalAssets := GetAssetsPaths(ga)
	globalJsSrc := globalAssets.AdminJsSrc
	globalCssHref := globalAssets.AdminCssHref

	var themeJsSrc, themeCssHref string
	if themeAPI.ThemesAPI.AdminTheme != nil {
		themeJsSrc = themeAPI.HttpAPI.Helpers().AdminAssetPath(themeAPI.ThemesAPI.AdminTheme.JsFile)
		themeCssHref = themeAPI.HttpAPI.Helpers().AdminAssetPath(themeAPI.ThemesAPI.AdminTheme.CssFile)
	}

	pluginGlobalJsSrc := pluginAPI.HttpAPI.Helpers().AdminAssetPath("global.js")
	pluginGlobalCssHref := pluginAPI.HttpAPI.Helpers().AdminAssetPath("global.css")
	pageJsSrc := pluginAPI.HttpAPI.Helpers().AdminAssetPath(v.Assets.JsFile)
	pageCssHref := pluginAPI.HttpAPI.Helpers().AdminAssetPath(v.Assets.CssFile)

	return themes.AdminAssets{
		GlobalCssHref:       globalCssHref,
		GlobalJsSrc:         globalJsSrc,
		ThemeCssHref:        themeCssHref,
		ThemeJsSrc:          themeJsSrc,
		PluginGlobalCssHref: pluginGlobalCssHref,
		PluginGlobalJsSrc:   pluginGlobalJsSrc,
		PageCssHref:         pageCssHref,
		PageJsSrc:           pageJsSrc,
	}, nil
}

func GetPortalAssetsForPage(coreAPI *PluginApi, themeAPI *PluginApi, pluginAPI *PluginApi, v sdkapi.ViewPage, ga *GlobalAssets) (assets themes.PortalAssets, err error) {
	globalAssets := GetAssetsPaths(ga)
	globalJsSrc := globalAssets.PortalJsSrc
	globalCssHref := globalAssets.PortalCssHref

	var themeJsSrc, themeCssHref string
	if themeAPI.ThemesAPI.PortalTheme != nil {
		themeJsSrc = themeAPI.HttpAPI.Helpers().PortalAssetPath(themeAPI.ThemesAPI.PortalTheme.JsFile)
		themeCssHref = themeAPI.HttpAPI.Helpers().PortalAssetPath(themeAPI.ThemesAPI.PortalTheme.CssFile)
	}

	pluginGlobalJsSrc := pluginAPI.HttpAPI.Helpers().PortalAssetPath("global.js")
	pluginGlobalCssHref := pluginAPI.HttpAPI.Helpers().PortalAssetPath("global.css")
	pageJsSrc := pluginAPI.HttpAPI.Helpers().PortalAssetPath(v.Assets.JsFile)
	pageCssHref := pluginAPI.HttpAPI.Helpers().PortalAssetPath(v.Assets.CssFile)

	return themes.PortalAssets{
		GlobalCssHref:       globalCssHref,
		GlobalJsSrc:         globalJsSrc,
		ThemeCssHref:        themeCssHref,
		ThemeJsSrc:          themeJsSrc,
		PluginGlobalCssHref: pluginGlobalCssHref,
		PluginGlobalJsSrc:   pluginGlobalJsSrc,
		PageCssHref:         pageCssHref,
		PageJsSrc:           pageJsSrc,
	}, nil
}
