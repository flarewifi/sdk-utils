package api

import (
	"net/http"
	sdkapi "sdk/api"

	"github.com/gorilla/mux"
)

func NewHttpRouterInstance(api *PluginApi, mux *mux.Router, admin bool) *HttpRouterInstance {
	return &HttpRouterInstance{api, mux, admin}
}

type HttpRouterInstance struct {
	api   *PluginApi
	mux   *mux.Router
	admin bool
}

func (self *HttpRouterInstance) Router() *mux.Router {
	return self.mux
}

func (self *HttpRouterInstance) Get(path string, h http.HandlerFunc, mw ...func(next http.Handler) http.Handler) sdkapi.IHttpRoute {
	finalHandler := http.Handler(h)
	for i := len(mw) - 1; i >= 0; i-- {
		finalHandler = mw[i](finalHandler)
	}
	route := self.mux.Handle(path, finalHandler).Methods("GET")
	return NewHttpRoute(self.api, route, self.admin)
}

func (self *HttpRouterInstance) Post(path string, h http.HandlerFunc, mw ...func(next http.Handler) http.Handler) sdkapi.IHttpRoute {
	finalHandler := http.Handler(h)
	for i := len(mw) - 1; i >= 0; i-- {
		finalHandler = mw[i](finalHandler)
	}
	route := self.mux.Handle(path, finalHandler).Methods("POST")
	return NewHttpRoute(self.api, route, self.admin)
}

func (self *HttpRouterInstance) Group(path string, fn func(sdkapi.IHttpRouterInstance)) {
	router := self.mux.PathPrefix(path).Subrouter()
	newrouter := NewHttpRouterInstance(self.api, router, self.admin)
	fn(newrouter)
}

func (self *HttpRouterInstance) Use(middlewares ...func(http.Handler) http.Handler) {
	for _, mw := range middlewares {
		self.mux.Use(mw)
	}
}
