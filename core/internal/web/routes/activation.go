package routes

import (
	"net/http"

	"core/internal/api"
	"core/internal/web/controllers"
	"core/internal/web/router"
)

func ActivationRoutes(g *api.CoreGlobals) {
	activationCtrl := controllers.NewActivationCtrl(g)

	router.RootRouter.HandleFunc(
		controllers.ActivationURL,
		activationCtrl.ActivationPage,
	).Methods(http.MethodGet).Name("activation:index")

	router.RootRouter.HandleFunc(
		controllers.ActivationURL+"/check",
		activationCtrl.CheckActivationStatus,
	).Methods(http.MethodPost).Name("activation:check")
}
