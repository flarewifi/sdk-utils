package controllers

import (
	"net/http"

	"core/internal/plugins"
)

func PaymentOptionsCtrl(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()
		// clnt, err := helpers.CurrentClient(g.ClientRegister, r)
		// if err != nil {
		// 	res.SendFlashMsg(w, "error", err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// methods := map[string]string{}
		// for _, opt := range g.PaymentsMgr.Options(clnt) {
		// 	methods[opt.Opt.OptName] = opt.VueRoutePath
		// }

		// res.Json(w, methods, http.StatusOK)

	}
}
