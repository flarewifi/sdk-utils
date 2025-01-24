package controllers

import (
	"net/http"
	sdkplugin "sdk/api"
)

func TestCtrl(api sdkplugin.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := api.Http().MuxVars(r)
		name := vars["name"]
		w.Write([]byte("Welcome " + name))
	}
}
