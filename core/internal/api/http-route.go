package api

import (
	sdkapi "sdk/api"

	"github.com/gorilla/mux"
)

func NewHttpRoute(api *PluginApi, r *mux.Route) *HttpRoute {
	return &HttpRoute{api, r}
}

type HttpRoute struct {
	api *PluginApi
	mux *mux.Route
}

func (self *HttpRoute) Queries(pairs ...string) sdkapi.IHttpRoute {
	self.mux.Queries(pairs...)
	return self
}

func (self *HttpRoute) Name(name sdkapi.PluginRouteName) sdkapi.IHttpRoute {
	muxname := self.api.HttpAPI.httpRouter.MuxRouteName(name)
	self.mux.Name(string(muxname))
	return self
}
