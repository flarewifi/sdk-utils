package api

import (
	"log"
	"path"
	"path/filepath"
	"strings"

	sdkapi "sdk/api"

	"core/internal/config"
	"core/internal/utils/flaretmpl"
	"core/internal/utils/pkg"
	"core/resources/views/themes"
)

func NewPluginUtils(api *PluginApi) *PluginUtils {
	return &PluginUtils{api}
}

type PluginUtils struct {
	api *PluginApi
}

func (self *PluginUtils) Translate(msgtype string, msgk string, pairs ...interface{}) string {
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

func (self *PluginUtils) GetAdminAssetsForPage(v sdkapi.ViewPage) (assets themes.AdminAssets) {
	// manifest := self.api.AssetsManifest
	globals := pkg.ReadGlobalAssetsManifest()
	globalJsSrc := self.api.CoreAPI.Http().Helpers().ResourcePath(path.Join("assets", "dist", globals.AdminJsFile))
	globalCssHref := self.api.CoreAPI.Http().Helpers().ResourcePath(path.Join("assets", "dist", globals.AdminCssFile))

	var themeJsSrc, themeCssHref string
	_, themesApi, err := self.api.PluginsMgrApi.GetAdminTheme()
	if err != nil {
		return
	} else {
		themeJsSrc, themeCssHref = themesApi.GetAdminAssets()
	}

	// var pluginGlobalJsSrc, pluginGlobalCssHref string
	// pluginGlobalJsFile, ok := manifest.AdminAssets.Scripts["global.js"]
	// if ok {
	// 	pluginGlobalJsSrc = self.api.HttpAPI.Helpers().AdminAssetPath(pluginGlobalJsFile)
	// }
	pluginGlobalJsSrc := self.api.HttpAPI.Helpers().AdminAssetPath("global.js")
	// pluginGlobalCssFile, ok := manifest.AdminAssets.Styles["global.css"]
	// if ok {
	// 	pluginGlobalCssHref = self.api.HttpAPI.Helpers().AdminAssetPath(pluginGlobalCssFile)
	// }
	pluginGlobalCssHref := self.api.HttpAPI.Helpers().AdminAssetPath("global.css")

	// var pageJsSrc, pageCssHref string
	// jsFile, ok := manifest.AdminAssets.Scripts[v.Assets.JsFile]
	// if ok {
	// 	pageJsSrc = self.api.HttpAPI.Helpers().AdminAssetPath(jsFile)
	// }
	pageJsSrc := self.api.HttpAPI.Helpers().AdminAssetPath(v.Assets.JsFile)

	// cssFile, ok := manifest.AdminAssets.Styles[v.Assets.CssFile]
	// if ok {
	// 	pageCssHref = self.api.HttpAPI.Helpers().AdminAssetPath(v.Assets.CssFile)
	// }
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
	}
}

func (self *PluginUtils) GetPortalAssetsForPage(v sdkapi.ViewPage) (assets themes.PortalAssets) {

	// manifest := self.api.AssetsManifest
	globals := pkg.ReadGlobalAssetsManifest()
	globalJsSrc := self.api.CoreAPI.Http().Helpers().ResourcePath(filepath.Join("assets", "dist", globals.PortalJsFile))
	globalCssHref := self.api.CoreAPI.Http().Helpers().ResourcePath(filepath.Join("assets", "dist", globals.PortalCssFile))

	var themeJsSrc, themeCssHref string
	_, themesApi, err := self.api.PluginsMgrApi.GetPortalTheme()
	if err != nil {
		return
	} else {
		themeJsSrc, themeCssHref = themesApi.GetPortalAssets()
	}

	// var pluginGlobalJsSrc, pluginGlobalCssHref string
	// pluginGlobalJsFile, ok := manifest.PortalAssets.Scripts["global.js"]
	// if ok {
	// 	pluginGlobalJsSrc = self.api.HttpAPI.Helpers().AssetPath(pluginGlobalJsFile)
	// }
	pluginGlobalJsSrc := self.api.HttpAPI.Helpers().PortalAssetPath("global.js")

	// pluginGlobalCssFile, ok := manifest.PortalAssets.Styles["global.css"]
	// if ok {
	// 	pluginGlobalCssHref = self.api.HttpAPI.Helpers().AssetPath(pluginGlobalCssFile)
	// }
	pluginGlobalCssHref := self.api.HttpAPI.Helpers().PortalAssetPath("global.css")

	// var pageJsSrc, pageCssHref string
	// jsFile, ok := manifest.PortalAssets.Scripts[v.Assets.JsFile]
	// if ok {
	// 	pageJsSrc = self.api.HttpAPI.Helpers().AssetPath(jsFile)
	// }

	// cssFile, ok := manifest.PortalAssets.Styles[v.Assets.CssFile]
	// if ok {
	// 	pageCssHref = self.api.HttpAPI.Helpers().AssetPath(cssFile)
	// }
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
	}
}
