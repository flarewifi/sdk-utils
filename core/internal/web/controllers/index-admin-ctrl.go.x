package controllers

import (
	"net/http"
	"path/filepath"

	"github.com/goccy/go-json"

	"core/internal/config"
	"core/internal/plugins"
	"core/internal/utils/assets"
	sse "core/internal/utils/sse"
	webutil "core/internal/utils/web"
	"core/internal/web/response"
)

func AdminIndexPage(g *plugins.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg, err := config.ReadThemesConfig()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		themePkg := cfg.Admin
		themePlugin, ok := g.PluginMgr.FindByPkg(themePkg)
		if !ok {
			http.Error(w, "Invalid admin theme", 500)
			return
		}

		themeApi := themePlugin.Themes().(*plugins.ThemesApi)
		routes, err := webutil.GetAdminRoutesData(g, themeApi)
		if err != nil {
			response.ErrorHtml(w, err.Error())
			return
		}

		appcfg, err := config.ReadApplicationConfig()
		if err != nil {
			response.ErrorHtml(w, err.Error())
			return
		}

		routesJson, err := json.Marshal(routes)
		if err != nil {
			response.ErrorHtml(w, err.Error())
			return
		}

		ssePath := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:sse")

		jsFiles := []assets.AssetWithData{
			// libs
			{File: g.CoreAPI.Utl.Resource("assets/libs/nprogress-0.2.0.js")},
			{File: g.CoreAPI.Utl.Resource("assets/libs/toastify-1.12.0.min.js")},
			{File: g.CoreAPI.Utl.Resource("assets/libs/promise-polyfill.min.js")},
			{File: g.CoreAPI.Utl.Resource("assets/libs/event-source.polyfill.min.js")},
			{File: g.CoreAPI.Utl.Resource("assets/libs/vue-2.7.16.min.js")},
			{File: g.CoreAPI.Utl.Resource("assets/libs/vue-router-3.6.5.min.js")},
			{File: g.CoreAPI.Utl.Resource("assets/libs/vue-demi-0.14.6.js")},
			{File: g.CoreAPI.Utl.Resource("assets/libs/vuelidate-core.min.js")},
			{File: g.CoreAPI.Utl.Resource("assets/libs/vuelidate-validators.min.js")},

			// app
			{File: g.CoreAPI.Utl.Resource("assets/services/require-config.js")},
			{File: g.CoreAPI.Utl.Resource("assets/services/basic-http.js")},
			{File: g.CoreAPI.Utl.Resource("assets/services/flare.vueLazyLoad.js")},
			{File: g.CoreAPI.Utl.Resource("assets/services/flare.events.js"), Data: ssePath},
			{File: g.CoreAPI.Utl.Resource("assets/services/flare.utils.js")},
			{File: g.CoreAPI.Utl.Resource("assets/services/flare.http.js")},
			{File: g.CoreAPI.Utl.Resource("assets/services/flare.notify.js")},
			{File: g.CoreAPI.Utl.Resource("assets/services/flare.forms.js"), Data: themeApi},
			{File: g.CoreAPI.Utl.Resource("assets/admin/router.js"), Data: string(routesJson)},
		}

		adminAssets := themeApi.GetAdminThemeAssets()
		for _, path := range adminAssets.Scripts {
			file := themePlugin.Resource(filepath.Join("assets", path))
			jsFiles = append(jsFiles, assets.AssetWithData{File: file})
		}

		cssFiles := []assets.AssetWithData{
			{File: g.CoreAPI.Utl.Resource("assets/libs/nprogress-0.2.0.css")},
			{File: g.CoreAPI.Utl.Resource("assets/libs/toastify-1.12.0.min.css")},
		}

		for _, path := range adminAssets.Styles {
			file := themePlugin.Resource(filepath.Join("assets", path))
			cssFiles = append(cssFiles, assets.AssetWithData{File: file})
		}

		jsBundle, err := g.CoreAPI.Utl.BundleAssetsWithHelper(w, r, jsFiles...)
		if err != nil {
			response.ErrorHtml(w, err.Error())
			return
		}

		cssBundle, err := g.CoreAPI.Utl.BundleAssetsWithHelper(w, r, cssFiles...)
		if err != nil {
			response.ErrorHtml(w, err.Error())
			return
		}

		vdata := map[string]any{
			"Lang":          appcfg.Lang,
			"ThemesApi":     themeApi,
			"VendorScripts": jsBundle.PublicPath,
			"VendorStyles":  cssBundle.PublicPath,
		}

		api := g.CoreAPI
		api.Http().HttpResponse().AdminView(w, r, "index.html", vdata)
	})
}

func AdminSseHandler(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := sse.NewSocket(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		acct, err := g.CoreAPI.HttpAPI.Auth().CurrentAcct(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sse.AddSocket(acct.Username(), s)
		s.Listen()
	}
}
