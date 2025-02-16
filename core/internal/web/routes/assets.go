package routes

import (
	"net/http"

	"core/internal/api"
	webutils "core/internal/utils/web"
	"core/internal/web/controllers"
	"core/internal/web/middlewares"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func AssetsRoutes(g *api.CoreGlobals) {
	cacheMw := middlewares.CacheResponse(365)
	assetsCtrl := controllers.NewAssetsCtrl(g)

	webutils.RootRouter.Handle("/favicon.ico", cacheMw(http.HandlerFunc(assetsCtrl.GetFavicon)))

	allPlugins := g.PluginMgr.All()
	for _, p := range allPlugins {
		resourcesDir := p.Resource("")
		fs := http.FileServer(http.Dir(resourcesDir))
		prefix := p.Http().Helpers().ResourcePath("")
		fileserver := cacheMw(http.StripPrefix(prefix, fs))
		webutils.RootRouter.PathPrefix(prefix).Handler(fileserver)
	}

	// set public static files
	publicDir := sdkutils.PathPublicDir
	fs := http.FileServer(http.Dir(publicDir))
	prefix := "/public"
	fileserver := cacheMw(http.StripPrefix(prefix, fs))
	webutils.RootRouter.PathPrefix(prefix).Handler(fileserver)
}

func CoreAssets(g *api.CoreGlobals) {
	cacheMw := middlewares.CacheResponse(365)
	resourcesDir := g.CoreAPI.Resource("")
	fs := http.FileServer(http.Dir(resourcesDir))
	prefix := g.CoreAPI.Http().Helpers().ResourcePath("")
	fileserver := cacheMw(http.StripPrefix(prefix, fs))
	webutils.RootRouter.PathPrefix(prefix).Handler(fileserver)
}
