package api

import (
	"log"
	sdkapi "sdk/api"
	"strings"

	"github.com/gorilla/mux"
)

func NewHttpRoute(api *PluginApi, r *mux.Route, admin bool) *HttpRoute {
	return &HttpRoute{api, r, admin}
}

type HttpRoute struct {
	api   *PluginApi
	mux   *mux.Route
	admin bool
}

func (self *HttpRoute) Queries(pairs ...string) sdkapi.IHttpRoute {
	self.mux.Queries(pairs...)
	return self
}

func (self *HttpRoute) Name(name sdkapi.PluginRouteName) sdkapi.IHttpRoute {
	if self.admin {
		if !strings.HasPrefix(string(name), RouteNameAdminPrefix) {
			log.Fatalf(`The admin route name "%s" must have a prefix "%s", for example: "%sdashboard.index"`, name, RouteNameAdminPrefix, RouteNameAdminPrefix)
		}
	} else {
		if strings.HasPrefix(string(name), RouteNameAdminPrefix) {
			log.Fatalf(`Route name "%s" must not have a prefix "%s"`, name, RouteNameAdminPrefix)
		}
	}

	muxname := self.api.HttpAPI.httpRouter.MuxRouteName(name)
	self.mux.Name(string(muxname))
	return self
}
