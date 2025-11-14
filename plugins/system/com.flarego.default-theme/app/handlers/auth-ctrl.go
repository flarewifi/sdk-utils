package handlers

import (
	"net/http"
	sdkapi "sdk/api"
)

func AdminAuthenticateCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			api.Http().Response().FlashMsg(w, r, api.Translate("error", "invalid_form_data"), sdkapi.FlashMsgError)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		acct, err := api.Http().Auth().Authenticate(username, password)
		if err != nil {
			api.Http().Response().FlashMsg(w, r, api.Translate("error", "invalid_credentials"), sdkapi.FlashMsgError)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		api.Http().Auth().SignIn(w, acct)
		api.Http().Response().FlashMsg(w, r, "Logged in successfully", sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

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
