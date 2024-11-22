package plugins

import (
	"net/http"

	sdkhttp "sdk/api/http"

	"github.com/gorilla/mux"
)

func NewHttpRouterInstance(api *PluginApi, mux *mux.Router) *HttpRouterInstance {
	return &HttpRouterInstance{api, mux}
}

type HttpRouterInstance struct {
	api *PluginApi
	mux *mux.Router
}

func (self *HttpRouterInstance) Router() *mux.Router {
	return self.mux
}

func (self *HttpRouterInstance) Get(path string, h http.HandlerFunc, mw ...func(next http.Handler) http.Handler) sdkhttp.IHttpRoute {
	finalHandler := http.Handler(h)
	for i := len(mw) - 1; i >= 0; i-- {
		finalHandler = mw[i](finalHandler)
	}
	route := self.mux.Handle(path, finalHandler).Methods("GET")
	return NewHttpRoute(self.api, route)
}

func (self *HttpRouterInstance) Post(path string, h http.HandlerFunc, mw ...func(next http.Handler) http.Handler) sdkhttp.IHttpRoute {
	finalHandler := http.Handler(h)
	for i := len(mw) - 1; i >= 0; i-- {
		finalHandler = mw[i](finalHandler)
	}
	route := self.mux.Handle(path, finalHandler).Methods("POST")
	return NewHttpRoute(self.api, route)
}

func (self *HttpRouterInstance) Group(path string, fn func(sdkhttp.IHttpRouterInstance)) {
	router := self.mux.PathPrefix(path).Subrouter()
	newrouter := NewHttpRouterInstance(self.api, router)
	fn(newrouter)
}

func (self *HttpRouterInstance) Use(middlewares ...func(http.Handler) http.Handler) {
	for _, mw := range middlewares {
		self.mux.Use(mw)
	}
}
