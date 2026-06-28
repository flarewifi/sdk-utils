package themes

import (
	"net/http"

	sdkapi "sdk/api"
)

// SetupRoutes registers the Devkit theme's own admin login route. Like the
// other product themes, the login is served over HTTPS only via the plugin's
// own auth:login route on the HTTPS-only HttpRouter.
func SetupRoutes(api sdkapi.IPluginApi) {
	httpsR := api.Http().Router().HttpRouter(&sdkapi.HttpRouterOpts{HttpsOnly: true})
	httpsR.Post("/login", adminAuthenticateCtrl(api)).Name("auth:login")
}

// adminAuthenticateCtrl authenticates an admin sign-in submitted from the
// captive portal login page.
func adminAuthenticateCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			api.Http().Response().FlashMsg(w, r, api.Translate("error", "Invalid form data"), sdkapi.FlashMsgError)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		acct, err := api.Http().Auth().Authenticate(username, password)
		if err != nil {
			api.Http().Response().FlashMsg(w, r, api.Translate("error", "Invalid credentials"), sdkapi.FlashMsgError)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		api.Http().Auth().SignIn(w, acct)
		api.Http().Response().FlashMsg(w, r, api.Translate("info", "Logged in successfully"), sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}
