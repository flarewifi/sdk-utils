package router

import (
	"fmt"
	"sync"

	sdkapi "sdk/api"

	"github.com/gorilla/mux"
)

const (
	NotFoundRoute string = "/404"
)

var (
	RootRouter         *mux.Router
	BootingRouter      *mux.Router
	PluginRouter       *mux.Router
	HttpsPluginRouter  *mux.Router
	AdminRouter        *mux.Router
	StaticPluginRouter *mux.Router
	StaticAdminRouter  *mux.Router

	// routeRegistry maps MuxRouteName → *mux.Route for O(1) lookup.
	routeRegistry sync.Map
)

func init() {
	RootRouter = mux.NewRouter().StrictSlash(true)
	BootingRouter = mux.NewRouter().StrictSlash(true)
	PluginRouter = RootRouter.PathPrefix("/p").Subrouter()
	AdminRouter = RootRouter.PathPrefix("/admin").Subrouter()
	StaticPluginRouter = RootRouter.PathPrefix("/p/static").Subrouter()
	// HTTPS-only plugin routes live under /p/https. Like /p/static, this is a
	// sibling sub-prefix of /p; gorilla resolves it via match-backtracking (try
	// /p first, then fall through here when no /p subroute matches).
	HttpsPluginRouter = RootRouter.PathPrefix("/p/https").Subrouter()
	StaticAdminRouter = RootRouter.PathPrefix("/admin/static").Subrouter()
}

// RegisterRoute stores a named route in the registry. Called by HttpRoute.Name().
func RegisterRoute(muxname sdkapi.MuxRouteName, route *mux.Route) {
	routeRegistry.Store(muxname, route)
}

// UrlForRoute resolves a URL for a plugin route by its namespaced MuxRouteName.
// It only searches the plugin route registry (sync.Map). Core routes registered
// directly on RootRouter via gorilla's .Name() must use RootRouter.Get() directly.
func UrlForRoute(muxname sdkapi.MuxRouteName, pairs ...string) (string, error) {
	route := FindRoute(muxname)
	if route != nil {
		if url, err := route.URL(pairs...); err == nil {
			return url.EscapedPath(), nil
		}
	}
	return "", fmt.Errorf("Route name not found: \"%s\"", muxname)
}

// FindRoute looks up a plugin route by its namespaced MuxRouteName.
// It only searches the plugin route registry (sync.Map); it does NOT search
// gorilla's native router. Core routes registered directly on RootRouter via
// gorilla's .Name() must use RootRouter.Get() directly.
func FindRoute(muxname sdkapi.MuxRouteName) *mux.Route {
	if v, ok := routeRegistry.Load(muxname); ok {
		return v.(*mux.Route)
	}
	return nil
}
