package routes

import (
	"core/internal/api"
	"core/internal/web/controllers"
	"core/internal/web/middlewares"
	"core/internal/web/router"
	"net/http"
)

func BootRoutes(g *api.CoreGlobals) {
	bootCtrl := controllers.NewBootCtrl(g)

	r := router.BootingRouter
	// Always land the booting page on the machine's LAN IP, never a domain name:
	// if a client reaches the machine by hostname during boot, normalize the host
	// to the LAN IP first so the page (and its /boot/status, /boot/progress polls)
	// stay on the IP. Runs before the boot-redirect middleware so the host is fixed
	// up before the path is rewritten to /boot. No-op in dev builds (no LAN concept).
	r.Use(middlewares.RedirectToLanIP(g.CoreAPI))
	r.Use(bootCtrl.Middleware)
	r.HandleFunc(controllers.BootURL, bootCtrl.BootPage).Methods(http.MethodGet)

	// Live boot timeline (JSON) the booting page polls to render progress.
	r.HandleFunc(controllers.BootProgressURL, bootCtrl.BootProgress).Methods(http.MethodGet)

	// GET "/boot/status" NOT OK
	r.HandleFunc(controllers.BootStatusURL, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusExpectationFailed)
	}).Methods(http.MethodGet)

	// Catch-all for any other path. This MUST be a registered route, not
	// r.NotFoundHandler: gorilla/mux only runs r.Use() middleware for *matched*
	// routes, so the NotFoundHandler bypasses RedirectToLanIP and bootCtrl.Middleware
	// entirely — an unknown path (e.g. a reloaded /admin page) would land on /boot
	// WITHOUT the redirect_to param. Registering it as a route lets the middleware
	// chain run, so bootCtrl.Middleware captures the original URL into redirect_to.
	// Only non-GET / asset requests slip past the middleware to this handler; they
	// have nothing to remember, so they just fall through to the boot page.
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, controllers.BootURL, http.StatusFound)
	})
}
