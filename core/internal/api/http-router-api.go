package api

import (
	"fmt"
	"log"
	"net/http"

	"core/db"
	"core/internal/sessmgr"
	"core/internal/web/middlewares"
	"core/internal/web/router"
	sdkapi "sdk/api"

	"github.com/gorilla/mux"
)

const (
	RouteNameAdminPrefix = "admin:"
	RouteNameAdminSSE    = "admin:sse"
)

type HttpRouterApi struct {
	api               *PluginApi
	adminRouter       *HttpRouterInstance
	pluginRouter      *HttpRouterInstance
	portalMiddlewares []func(http.Handler) http.Handler
}

func NewHttpRouterApi(api *PluginApi, db *db.Database, clnt *sessmgr.ClientRegister) *HttpRouterApi {
	prefix := fmt.Sprintf("/%s/%s", api.info.Package, api.info.Version)
	pluginMux := router.PluginRouter.PathPrefix(prefix).Subrouter()
	adminMux := router.AdminRouter.PathPrefix(prefix).Subrouter()

	pluginRouter := &HttpRouterInstance{api, pluginMux, false}
	adminRouter := &HttpRouterInstance{api, adminMux, true}

	return &HttpRouterApi{
		api:               api,
		adminRouter:       adminRouter,
		pluginRouter:      pluginRouter,
		portalMiddlewares: []func(http.Handler) http.Handler{},
	}
}

func (self *HttpRouterApi) Initialize() {
	self.adminRouter.Use(middlewares.HTTPSRedirect())
	self.adminRouter.Use(middlewares.AdminAuth(self.api.CoreAPI))
}

func (self *HttpRouterApi) AdminRouter() sdkapi.IHttpRouterInstance {
	return self.adminRouter
}

func (self *HttpRouterApi) PluginRouter() sdkapi.IHttpRouterInstance {
	return self.pluginRouter
}

func (self *HttpRouterApi) Use(middleware ...func(http.Handler) http.Handler) {
	for _, mw := range middleware {
		router.RootRouter.Use(mux.MiddlewareFunc(mw))
	}
}

func (self *HttpRouterApi) MuxRouteName(name sdkapi.PluginRouteName) sdkapi.MuxRouteName {
	muxname := fmt.Sprintf("%s#%s", self.api.info.Package, string(name))
	return sdkapi.MuxRouteName(muxname)
}

func (self *HttpRouterApi) UrlForMuxRoute(muxname sdkapi.MuxRouteName, pairs ...string) string {
	route := router.RootRouter.Get(string(muxname))
	if route == nil {
		log.Println("Error: route not found for " + string(muxname))
		return "Error: route not found for " + string(muxname)
	}

	url, err := route.URL(pairs...)
	if err != nil {
		log.Println("Error: " + err.Error())
		return ""
	}

	return url.String()
}

func (self *HttpRouterApi) UrlForRoute(name sdkapi.PluginRouteName, pairs ...string) string {
	muxname := self.MuxRouteName(name)
	return self.UrlForMuxRoute(muxname, pairs...)
}

func (self *HttpRouterApi) UrlForPkgRoute(pkg string, name string, pairs ...string) string {
	otherPkg, ok := self.api.PluginsMgrApi.FindByPkg(pkg)
	if !ok {
		return ""
	}
	return otherPkg.Http().Helpers().UrlForRoute(name, pairs...)
}

func (self *HttpRouterApi) UseForPortal(middlewares ...func(http.Handler) http.Handler) {
	self.portalMiddlewares = append(self.portalMiddlewares, middlewares...)
}

func (self *HttpRouterApi) GetPortalMiddlewares() []func(http.Handler) http.Handler {
	return self.portalMiddlewares
}
