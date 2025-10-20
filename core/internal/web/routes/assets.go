package routes

import (
	"net/http"

	"core/internal/api"
	webutils "core/internal/utils/web"
	"core/internal/web/middlewares"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func AssetsRoutes(g *api.CoreGlobals) {
	cacheMw := middlewares.CacheResponse(365)

	allPlugins := g.PluginMgr.All()
	for _, p := range allPlugins {
		h := p.Http().Helpers().(*api.HttpHelpers)
		distDir := p.Resource("assets/dist")
		distFs := http.FileServer(http.Dir(distDir))
		distPrefix := h.DistPath("")
		distServer := cacheMw(http.StripPrefix(distPrefix, distFs))
		webutils.RootRouter.PathPrefix(distPrefix).Handler(distServer)

		pubDir := p.Resource("assets/public")
		pubFs := http.FileServer(http.Dir(pubDir))
		pubPrefix := h.PublicPath("")
		pubServer := cacheMw(http.StripPrefix(pubPrefix, pubFs))
		webutils.RootRouter.PathPrefix(pubPrefix).Handler(pubServer)
	}

	// set public static files
	publicDir := sdkutils.PathPublicDir
	fs := http.FileServer(http.Dir(publicDir))
	prefix := "/public"
	fileserver := cacheMw(http.StripPrefix(prefix, fs))
	webutils.RootRouter.PathPrefix(prefix).Handler(fileserver)
}

func CoreAssets(g *api.CoreGlobals) {
	h := g.CoreAPI.Http().Helpers().(*api.HttpHelpers)
	cacheMw := middlewares.CacheResponse(365)
	distDir := g.CoreAPI.Resource("assets/dist")
	distFs := http.FileServer(http.Dir(distDir))
	distPrefix := h.DistPath("")
	distServer := cacheMw(http.StripPrefix(distPrefix, distFs))
	webutils.BootingRouter.PathPrefix(distPrefix).Handler(distServer)

	pubDir := g.CoreAPI.Resource("assets/public")
	pubFs := http.FileServer(http.Dir(pubDir))
	pubPrefix := h.PublicPath("")
	pubServer := cacheMw(http.StripPrefix(pubPrefix, pubFs))
	webutils.BootingRouter.PathPrefix(distPrefix).Handler(pubServer)
}
