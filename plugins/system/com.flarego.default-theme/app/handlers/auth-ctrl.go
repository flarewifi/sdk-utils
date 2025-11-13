package handlers

import (
	"net/http"
	sdkapi "sdk/api"
)

func LogoutCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := api.Http().Auth().SignOut(w); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		api.Http().Response().FlashMsg(w, r, "Logged out successfully", sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}
