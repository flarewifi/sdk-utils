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
		controllers.ActivationURL+"/status",
		activationCtrl.GetActivationStatus,
	).Methods(http.MethodGet).Name("activation:status")

	router.RootRouter.HandleFunc(
		controllers.ActivationURL+"/validate",
		activationCtrl.ValidateActivation,
	).Methods(http.MethodPost).Name("activation:validate")
}
