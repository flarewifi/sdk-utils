package api

import (
	"log"
	"path/filepath"
	"strings"

	sdkapi "sdk/api"

	"core/internal/config"
	"core/internal/utils/flaretmpl"
	"core/internal/utils/plugins"
	"core/resources/views/themes"
)

func NewPluginUtils(api *PluginApi) *PluginUtils {
	return &PluginUtils{api}
}

type PluginUtils struct {
	api *PluginApi
}

func (self *PluginUtils) Translate(msgtype string, msgk string, pairs ...any) string {
	if len(pairs)%2 != 0 {
		log.Printf("Translate pairs: %+v", pairs)
		return "Invalid number of translation params."
	}

	trnsdir := self.Resource("translations")
	appcfg, _ := config.ReadApplicationConfig()

	f := filepath.Join(trnsdir, appcfg.Lang, msgtype, msgk+".txt")
	tmpl, err := flaretmpl.GetTextTemplate(f)
	if err != nil {
		log.Println("Warning: Translation file not found: ", f)
		return msgk
	}

	vdata := map[interface{}]interface{}{}
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i]
		value := pairs[i+1]
		vdata[key] = value
	}

	var output strings.Builder
	err = tmpl.Execute(&output, vdata)
	if err != nil {
		log.Println("Error executing translation template "+f, err)
		return msgk
	}

	s := output.String()

	return strings.TrimSpace(s)
}

func (self *PluginUtils) Resource(path string) string {
	return filepath.Join(self.api.dir, "resources", path)
}

func (self *PluginUtils) GetAdminAssetsForPage(v sdkapi.ViewPage) (assets themes.AdminAssets, err error) {
	_, themesApi, err := self.api.PluginsMgrApi.GetAdminTheme()
	if err != nil {
		return
	}

	globals := plugins.ReadGlobalAssetsManifest()
	h := self.api.CoreAPI.HttpAPI.Helpers().(*HttpHelpers)
	// globalJsSrc := self.api.CoreAPI.Http().Helpers().ResourcePath(path.Join("assets", "dist", globals.AdminJsFile))
	// globalCssHref := self.api.CoreAPI.Http().Helpers().ResourcePath(path.Join("assets", "dist", globals.AdminCssFile))
	globalJsSrc := h.DistPath(globals.AdminJsFile)
	globalCssHref := h.DistPath(globals.AdminCssFile)

	var themeJsSrc, themeCssHref string
	if themesApi.AdminTheme != nil {
		themeJsSrc = themesApi.api.HttpAPI.Helpers().AdminAssetPath(themesApi.AdminTheme.JsFile)
		themeCssHref = themesApi.api.HttpAPI.Helpers().AdminAssetPath(themesApi.AdminTheme.CssFile)
	}

	pluginGlobalJsSrc := self.api.HttpAPI.Helpers().AdminAssetPath("global.js")
	pluginGlobalCssHref := self.api.HttpAPI.Helpers().AdminAssetPath("global.css")
	pageJsSrc := self.api.HttpAPI.Helpers().AdminAssetPath(v.Assets.JsFile)
	pageCssHref := self.api.HttpAPI.Helpers().AdminAssetPath(v.Assets.CssFile)

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

func (self *PluginUtils) GetPortalAssetsForPage(v sdkapi.ViewPage) (assets themes.PortalAssets, err error) {
	_, themesApi, err := self.api.PluginsMgrApi.GetPortalTheme()
	if err != nil {
		return
	}

	globals := plugins.ReadGlobalAssetsManifest()
	h := self.api.CoreAPI.HttpAPI.Helpers().(*HttpHelpers)
	// globalJsSrc := self.api.CoreAPI.Http().Helpers().ResourcePath(filepath.Join("assets", "dist", globals.PortalJsFile))
	// globalCssHref := self.api.CoreAPI.Http().Helpers().ResourcePath(filepath.Join("assets", "dist", globals.PortalCssFile))
	globalJsSrc := h.DistPath(globals.PortalJsFile)
	globalCssHref := h.DistPath(globals.PortalCssFile)

	var themeJsSrc, themeCssHref string
	if themesApi.PortalTheme != nil {
		themeJsSrc = themesApi.api.HttpAPI.Helpers().PortalAssetPath(themesApi.PortalTheme.JsFile)
		themeCssHref = themesApi.api.HttpAPI.Helpers().PortalAssetPath(themesApi.PortalTheme.CssFile)
	}

	pluginGlobalJsSrc := self.api.HttpAPI.Helpers().PortalAssetPath("global.js")
	pluginGlobalCssHref := self.api.HttpAPI.Helpers().PortalAssetPath("global.css")

	pageJsSrc := self.api.HttpAPI.Helpers().PortalAssetPath(v.Assets.JsFile)
	pageCssHref := self.api.HttpAPI.Helpers().PortalAssetPath(v.Assets.CssFile)

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
