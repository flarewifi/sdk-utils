package routes

import (
	"net/http"
	"path"
	"path/filepath"

	"core/internal/api"
	"core/internal/web/controllers"
	"core/internal/web/middlewares"
	"core/internal/web/router"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"

	"github.com/gorilla/mux"
)

func PluginAssets(g *api.CoreGlobals) {
	allPlugins := g.PluginMgr.All()
	for _, p := range allPlugins {
		setupAssetsRoutes(router.RootRouter, p)
	}
}

func BootingAssets(g *api.CoreGlobals) {
	p := g.CoreAPI
	setupAssetsRoutes(router.BootingRouter, p)
}

func GlobalAssets(g *api.CoreGlobals) {
	r := router.RootRouter
	cacheMw := middlewares.CacheResponse(365) // cache for 1 year
	assets := api.GetAssetsPaths(g.GlobalAssets)

	r.PathPrefix(assets.AdminJsSrc).Handler(cacheMw(controllers.GlobalAdminJsCtrl(g)))
	r.PathPrefix(assets.AdminCssHref).Handler(cacheMw(controllers.GlobalAdminCssCtrl(g)))
	r.PathPrefix(assets.PortalJsSrc).Handler(cacheMw(controllers.GlobalPortalJsCtrl(g)))
	r.PathPrefix(assets.PortalCssHref).Handler(cacheMw(controllers.GlobalPortalCssCtrl(g)))
}

func setupAssetsRoutes(r *mux.Router, p sdkapi.IPluginApi) {
	cacheMw := middlewares.CacheResponse(365) // cache for 1 year
	h := p.Http().Helpers().(*api.HttpHelpers)

	distDir := p.Resource("assets/dist")
	distFs := http.FileServer(http.Dir(distDir))
	distPrefix := h.DistPath("")
	distServer := cacheMw(http.StripPrefix(distPrefix, distFs))
	r.PathPrefix(distPrefix).Handler(distServer)

	pubDir := p.Resource("assets/public")
	pubFs := http.FileServer(http.Dir(pubDir))
	pubPrefix := h.PublicPath("")
	pubServer := cacheMw(http.StripPrefix(pubPrefix, pubFs))
	r.PathPrefix(pubPrefix).Handler(pubServer)

	// Storage files route
	storageDir := filepath.Join(sdkutils.PathPluginStorageDir, p.Info().Package)
	if sdkutils.FsExists(storageDir) {
		storageFs := http.FileServer(http.Dir(storageDir))
		storagePrefix := path.Join("/storage/plugin", p.Info().Package)
		storageMw := middlewares.CacheResponse(7) // 7 days cache
		storageServer := storageMw(http.StripPrefix(storagePrefix, storageFs))
		r.PathPrefix(storagePrefix + "/").Handler(storageServer)
	}
}
