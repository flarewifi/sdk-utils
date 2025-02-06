package plugins

import (
	"log"
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

func (self *PluginUtils) GetAdminAssetsForPage(v sdkapi.ViewPage) (assets sdkapi.PageAssets) {

	var themeJsSrc, themeCssHref string
	_, themesApi, err := self.api.PluginsMgrApi.GetAdminTheme()
	if err != nil {
		return
	} else {
		themeJsSrc, themeCssHref = themesApi.GetAdminAssets()
	}

	globals := pkg.ReadGlobalAssetsManifest()
	globalJsSrc := self.api.CoreAPI.Http().Helpers().AssetPath(globals.AdminJsFile)
	globalCssHref := self.api.CoreAPI.Http().Helpers().AssetPath(globals.AdminCssFile)

	manifest := self.api.AssetsManifest

	var pageJsSrc string
	jsFile, ok := manifest.AdminAssets.Scripts[v.Assets.JsFile]
	if ok {
		pageJsSrc = self.api.HttpAPI.Helpers().AssetPath(jsFile)
	}

	var pageCssHref string
	cssFile, ok := manifest.AdminAssets.Styles[v.Assets.CssFile]
	if ok {
		pageCssHref = self.api.HttpAPI.Helpers().AssetPath(cssFile)
	}

	assets.Stylesheets = themes.AdminThemeStylesheets(globalCssHref, themeCssHref, pageCssHref)
	assets.Javascripts = themes.AdminThemeScripts(globalJsSrc, themeJsSrc, pageJsSrc)

	return
}

func (self *PluginUtils) GetPortalAssetsForPage(v sdkapi.ViewPage) (assets sdkapi.PageAssets) {
	var themeJsSrc, themeCssHref string
	_, themesApi, err := self.api.PluginsMgrApi.GetPortalTheme()
	if err != nil {
		return
	} else {
		themeJsSrc, themeCssHref = themesApi.GetPortalAssets()
	}

	globals := pkg.ReadGlobalAssetsManifest()
	globalJsSrc := self.api.CoreAPI.Http().Helpers().AssetPath(globals.PortalJsFile)
	globalCssHref := self.api.CoreAPI.Http().Helpers().AssetPath(globals.PortalCssFile)

	manifest := self.api.AssetsManifest

	var pageJsSrc string
	jsFile, ok := manifest.PortalAssets.Scripts[v.Assets.JsFile]
	if ok {
		pageJsSrc = self.api.HttpAPI.Helpers().AssetPath(jsFile)
	}

	var pageCssHref string
	cssFile, ok := manifest.PortalAssets.Styles[v.Assets.CssFile]
	if ok {
		pageCssHref = self.api.HttpAPI.Helpers().AssetPath(cssFile)
	}

	assets.Stylesheets = themes.PortalThemeStylesheets(globalCssHref, themeCssHref, pageCssHref)
	assets.Javascripts = themes.PortalThemeJavascripts(globalJsSrc, themeJsSrc, pageJsSrc)

	return
}
