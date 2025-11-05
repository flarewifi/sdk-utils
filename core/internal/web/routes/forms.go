package routes

import (
	"core/internal/api"
	"core/internal/web/controllers"
	sdkapi "sdk/api"
)

func FormRoutes(g *api.CoreGlobals) {
	pRouter := g.CoreAPI.HttpAPI.Router().PluginRouter()

	pRouter.Group("/forms", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Post("/file/delete", controllers.DeleteFileCtrl(g)).Name("forms.file.delete")
	})
}
