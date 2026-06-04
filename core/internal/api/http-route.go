package api

import (
	"fmt"
	"strings"

	"core/internal/web/router"
	sdkapi "sdk/api"

	"github.com/gorilla/mux"
)

func NewHttpRoute(api *PluginApi, r *mux.Route, admin bool, static bool) *HttpRoute {
	return &HttpRoute{api: api, mux: r, admin: admin, static: static}
}

type HttpRoute struct {
	api    *PluginApi
	mux    *mux.Route
	admin  bool
	static bool
}

func (self *HttpRoute) Queries(pairs ...string) sdkapi.IHttpRoute {
	self.mux.Queries(pairs...)
	return self
}

func (self *HttpRoute) Name(name sdkapi.PluginRouteName) sdkapi.IHttpRoute {
	if self.admin {
		if !strings.HasPrefix(string(name), RouteNameAdminPrefix) {
			panic(fmt.Errorf(`The admin route name "%s" must have a prefix "%s", for example: "%sdashboard.index"`, name, RouteNameAdminPrefix, RouteNameAdminPrefix))
		}
	} else {
		if strings.HasPrefix(string(name), RouteNameAdminPrefix) {
			panic(fmt.Errorf(`Route name "%s" must not have a prefix "%s"`, name, RouteNameAdminPrefix))
		}
	}

	var muxname sdkapi.MuxRouteName
	if self.static {
		muxname = self.api.HttpAPI.httpRouter.staticMuxRouteName(name)
	} else {
		muxname = self.api.HttpAPI.httpRouter.MuxRouteName(name)
	}
	self.mux.Name(string(muxname))
	router.RegisterRoute(muxname, self.mux)
	return self
}
