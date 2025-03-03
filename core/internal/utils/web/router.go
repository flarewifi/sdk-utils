package webutil

import (
	"errors"
	"fmt"

	sdkapi "sdk/api"

	"github.com/gorilla/mux"
)

const (
	NotFoundRoute string = "/404"
)

var (
	RootRouter    *mux.Router
	BootingRouter *mux.Router
	PluginRouter  *mux.Router
	AssetsRouter  *mux.Router
)

func init() {
	RootRouter = mux.NewRouter().StrictSlash(true)
	BootingRouter = mux.NewRouter().StrictSlash(true)
	PluginRouter = RootRouter.PathPrefix("/p").Subrouter()
	AssetsRouter = RootRouter.PathPrefix("/assets").Subrouter()
}

func UrlForRoute(muxname sdkapi.MuxRouteName, pairs ...string) (string, error) {
	route := FindRoute(muxname)
	if route != nil {
		if url, err := route.URL(pairs...); err == nil {
			return url.EscapedPath(), nil
		}
	}
	return "", errors.New(fmt.Sprintf("Route name not found: \"%s\"", muxname))
}

func FindRoute(muxname sdkapi.MuxRouteName) *mux.Route {
	return RootRouter.Get(string(muxname))
}
