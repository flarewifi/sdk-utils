package routes

import (
	"core/internal/api"
	"core/internal/web/controllers"
	"core/internal/web/router"
	"net/http"
)

func BootRoutes(g *api.CoreGlobals) {
	bootCtrl := controllers.NewBootCtrl(g)

	r := router.BootingRouter
	r.Use(bootCtrl.Middleware)
	r.HandleFunc(controllers.BootURL, bootCtrl.BootPage).Methods(http.MethodGet)

	// GET "/boot/status" NOT OK
	r.HandleFunc(controllers.BootStatusURL, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusExpectationFailed)
	}).Methods(http.MethodGet)

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, controllers.BootURL, http.StatusFound)
	})
}
