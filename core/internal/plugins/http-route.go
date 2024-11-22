package plugins

import (
	sdkhttp "sdk/api/http"

	"github.com/gorilla/mux"
)

func NewHttpRoute(api *PluginApi, r *mux.Route) *HttpRoute {
	return &HttpRoute{api, r}
}

type HttpRoute struct {
	api *PluginApi
	mux *mux.Route
}

func (self *HttpRoute) Queries(pairs ...string) sdkhttp.IHttpRoute {
	self.mux.Queries(pairs...)
	return self
}

func (self *HttpRoute) Name(name sdkhttp.PluginRouteName) sdkhttp.IHttpRoute {
	muxname := self.api.HttpAPI.httpRouter.MuxRouteName(name)
	self.mux.Name(string(muxname))
	return self
}
