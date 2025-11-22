package api

import (
	"fmt"
	"log"
	"net/http"

	"core/db"
	"core/internal/connmgr"
	webutil "core/internal/utils/web"
	sdkapi "sdk/api"

	"github.com/gorilla/mux"
)

const (
	RouteNameAdminPrefix = "admin:"
	RouteNameAdminSSE    = "admin:sse"
)

type HttpRouterApi struct {
	api          *PluginApi
	adminRouter  *HttpRouterInstance
	pluginRouter *HttpRouterInstance
}

func NewHttpRouterApi(api *PluginApi, db *db.Database, clnt *connmgr.ClientRegister) *HttpRouterApi {
	prefix := fmt.Sprintf("/%s/%s", api.info.Package, api.info.Version)
	pluginMux := webutil.PluginRouter.PathPrefix(prefix).Subrouter()
	adminMux := pluginMux.PathPrefix("/admin").Subrouter()

	pluginRouter := &HttpRouterInstance{api, pluginMux, false}
	adminRouter := &HttpRouterInstance{api, adminMux, true}

	return &HttpRouterApi{api, adminRouter, pluginRouter}
}

func (self *HttpRouterApi) Initialize() {
	self.pluginRouter.Use(self.api.HttpAPI.middlewares.Device())
	self.adminRouter.Use(self.api.HttpAPI.middlewares.HTTPSRedirect())
	self.adminRouter.Use(self.api.HttpAPI.middlewares.AdminAuth())
	self.adminRouter.Use(self.api.HttpAPI.middlewares.TrackNav())
}

func (self *HttpRouterApi) AdminRouter() sdkapi.IHttpRouterInstance {
	return self.adminRouter
}

func (self *HttpRouterApi) PluginRouter() sdkapi.IHttpRouterInstance {
	return self.pluginRouter
}

func (self *HttpRouterApi) Use(middleware ...func(http.Handler) http.Handler) {
	for _, mw := range middleware {
		webutil.RootRouter.Use(mux.MiddlewareFunc(mw))
	}
}

func (self *HttpRouterApi) MuxRouteName(name sdkapi.PluginRouteName) sdkapi.MuxRouteName {
	muxname := fmt.Sprintf("%s#%s", self.api.info.Package, string(name))
	return sdkapi.MuxRouteName(muxname)
}

func (self *HttpRouterApi) UrlForMuxRoute(muxname sdkapi.MuxRouteName, pairs ...string) string {
	route := webutil.RootRouter.Get(string(muxname))
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
