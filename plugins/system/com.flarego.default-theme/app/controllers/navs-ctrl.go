package controllers

import (
	"net/http"

	sdkplugin "sdk/api/plugin"
)

func GetAdminNavs(api sdkplugin.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := api.Http().VueResponse()
		// acct, err := api.Http().Auth().CurrentAcct(r)
		// if err != nil {
		// 	res.Error(w, err.Error(), http.StatusUnauthorized)
		// 	return
		// }
		// navs := api.Http().GetAdminNavs(acct)
		// res.Json(w, navs, http.StatusOK)
	}
}
