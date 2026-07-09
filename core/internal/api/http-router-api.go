package api

import (
	"fmt"
	"net/http"

	"core/db"
	"core/internal/sessmgr"
	"core/internal/web/middlewares"
	"core/internal/web/router"
	sdkapi "sdk/api"
)

const (
	RouteNameAdminPrefix = "admin:"
	RouteNameAdminSSE    = "admin:sse"
)

type HttpRouterApi struct {
	api                *PluginApi
	adminRouter        *HttpRouterInstance
	httpRouter         *HttpRouterInstance
	httpsRouter        *HttpRouterInstance
	staticAdminRouter  *HttpRouterInstance
	staticPluginRouter *HttpRouterInstance
	portalMiddlewares  []func(http.Handler) http.Handler
}

func NewHttpRouterApi(api *PluginApi, db *db.Database, clnt *sessmgr.ClientRegister) *HttpRouterApi {
	prefix := fmt.Sprintf("/%s/%s", api.info.Package, api.info.Version)
	pluginMux := router.PluginRouter.PathPrefix(prefix).Subrouter()
	httpsMux := router.HttpsPluginRouter.PathPrefix(prefix).Subrouter()
	adminMux := router.AdminRouter.PathPrefix(prefix).Subrouter()

	staticPrefix := fmt.Sprintf("/%s", api.info.Package)
	staticPluginMux := router.StaticPluginRouter.PathPrefix(staticPrefix).Subrouter()
	staticAdminMux := router.StaticAdminRouter.PathPrefix(staticPrefix).Subrouter()

	httpRouter := NewHttpRouterInstance(api, pluginMux, false)
	httpsRouter := NewHttpRouterInstance(api, httpsMux, false)
	adminRouter := NewHttpRouterInstance(api, adminMux, true)
	staticPluginRouter := NewStaticHttpRouterInstance(api, staticPluginMux, false)
	staticAdminRouter := NewStaticHttpRouterInstance(api, staticAdminMux, true)

	return &HttpRouterApi{
		api:                api,
		adminRouter:        adminRouter,
		httpRouter:         httpRouter,
		httpsRouter:        httpsRouter,
		staticAdminRouter:  staticAdminRouter,
		staticPluginRouter: staticPluginRouter,
		portalMiddlewares:  []func(http.Handler) http.Handler{},
	}
}

func (self *HttpRouterApi) Initialize() {
	// HTTPS for admin pages is enforced globally by middlewares.ForceHTTPS on
	// RootRouter (see web.SetupAppRoutes); the admin routers only need auth here.
	self.adminRouter.Use(middlewares.AdminAuth(self.api.CoreAPI))
	self.staticAdminRouter.Use(middlewares.AdminAuth(self.api.CoreAPI))

	// The HttpsRouter is unconditionally HTTPS-only (no auth): both listeners
	// share RootRouter, so a per-instance guard is what keeps its routes off the
	// plain-HTTP listener.
	self.httpsRouter.Use(middlewares.RequireHTTPS())
}

// AdminRouter returns a router whose routes require an authenticated admin
// session. Admin routes are always served over HTTPS; when opts.Static is set,
// the returned router serves routes at /admin/static/{package}/{path} that
// persist across plugin version updates.
func (self *HttpRouterApi) AdminRouter(opts *sdkapi.AdminRouterOpts) sdkapi.IHttpRouterInstance {
	if opts != nil && opts.Static {
		return self.staticAdminRouter
	}
	return self.adminRouter
}

// HttpRouter returns a general-purpose plugin router with no authentication.
// The opts select a variant of the same router family:
//   - opts.Static:    routes persist across plugin version updates
//                     (/p/static/{package}/{path}).
//   - opts.HttpsOnly: routes are served only over HTTPS.
//   - nil / zero:     served over either scheme at the versioned plugin path.
//
// Static takes precedence over HttpsOnly when both are set.
func (self *HttpRouterApi) HttpRouter(opts *sdkapi.HttpRouterOpts) sdkapi.IHttpRouterInstance {
	if opts != nil {
		switch {
		case opts.Static:
			return self.staticPluginRouter
		case opts.HttpsOnly:
			return self.httpsRouter
		}
	}
	return self.httpRouter
}

// NOTE: there is deliberately NO Use() on this type. A middleware "for the
// plugin's router" must be registered on a scoped IHttpRouterInstance
// (HttpRouter/AdminRouter → Group/Use); the removed method mounted the
// middleware on the GLOBAL RootRouter, silently wrapping every route of every
// plugin. Portal-page middlewares go through UseForPortal.

func (self *HttpRouterApi) MuxRouteName(name sdkapi.PluginRouteName) sdkapi.MuxRouteName {
	muxname := fmt.Sprintf("%s#%s", self.api.info.Package, string(name))
	return sdkapi.MuxRouteName(muxname)
}

func (self *HttpRouterApi) staticMuxRouteName(name sdkapi.PluginRouteName) sdkapi.MuxRouteName {
	muxname := fmt.Sprintf("%s#static#%s", self.api.info.Package, string(name))
	return sdkapi.MuxRouteName(muxname)
}

func (self *HttpRouterApi) UrlForMuxRoute(muxname sdkapi.MuxRouteName, pairs ...string) string {
	route := router.FindRoute(muxname)
	if route == nil {
		return "Error: route not found for " + string(muxname)
	}

	url, err := route.URL(pairs...)
	if err != nil {
		return ""
	}

	return url.String()
}

func (self *HttpRouterApi) UrlForRoute(name sdkapi.PluginRouteName, pairs ...string) string {
	// Try versioned route first, fall back to static route automatically.
	muxname := self.MuxRouteName(name)
	if router.FindRoute(muxname) != nil {
		return self.UrlForMuxRoute(muxname, pairs...)
	}
	return self.UrlForMuxRoute(self.staticMuxRouteName(name), pairs...)
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

// ClaimPortalTraffic registers portal-traffic claim middlewares with the
// shared funnel registry (see middlewares.RegisterPortalClaim) — both funnel
// entry points (ForceHTTPS and the NotFoundHandler's RedirectToPortalDomain)
// run them before making any routing decision.
func (self *HttpRouterApi) ClaimPortalTraffic(claims ...func(next http.Handler) http.Handler) {
	middlewares.RegisterPortalClaim(claims...)
}

func (self *HttpRouterApi) GetPortalMiddlewares() []func(http.Handler) http.Handler {
	return self.portalMiddlewares
}
