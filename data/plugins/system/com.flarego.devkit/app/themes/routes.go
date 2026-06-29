package themes

import (
	"errors"
	"net/http"

	sdkapi "sdk/api"
)

// SetupRoutes registers the Devkit theme's OWN admin auth routes — both login
// and logout live in this plugin's namespace so the theme views resolve them
// without reaching into the core plugin's routes.
//
// Login is served over HTTPS only (credentials must never cross plain HTTP) and
// exposes BOTH a GET and a POST on /login:
//   - POST authenticates the submitted credentials.
//   - GET re-renders the login page. This matters because the HTTPS-only router
//     carries RequireHTTPS: an HTTP login POST is answered with a 302 to the
//     HTTPS URL, and browsers downgrade a 302-followed POST to a GET. Without a
//     GET handler that lands on the POST-only route as a 405; the GET re-renders
//     the form over HTTPS instead, and the next submit (now HTTPS) authenticates.
//
// Logout is registered on the auth-gated AdminRouter rather than the HTTPS-only
// router: it has no RequireHTTPS guard, so it is immune to the same 302→GET→405
// downgrade, and the admin session already guarantees HTTPS in production.
func SetupRoutes(api sdkapi.IPluginApi) {
	httpsR := api.Http().Router().HttpRouter(&sdkapi.HttpRouterOpts{HttpsOnly: true})
	httpsR.Get("/login", adminLoginPageCtrl(api)).Name("auth:login-page")
	httpsR.Post("/login", adminAuthenticateCtrl(api)).Name("auth:login")

	// Admin-router route names must carry the "admin:" prefix (enforced by core).
	// This is the devkit's OWN admin:auth:logout, namespaced under this plugin —
	// distinct from the core's identically-named route — so UrlForRoute resolves
	// it within this plugin and the layout's logout form gets a valid action.
	adminR := api.Http().Router().AdminRouter(nil)
	adminR.Post("/logout", adminLogoutCtrl(api)).Name("admin:auth:logout")
}

// adminLoginPageCtrl re-renders the theme's login page. It backs the GET on the
// HTTPS-only /login route so a login POST that gets bounced from HTTP to HTTPS
// (302, which downgrades the method to GET) lands on the form again instead of a
// 405. Already-authenticated admins are sent straight to the dashboard.
func adminLoginPageCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := api.Http().Auth().IsAuthenticated(r); err == nil {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}
		api.Http().Response().PortalView(w, r, loginViewPage(api, r, sdkapi.LoginPageData{}))
	}
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

// adminLogoutCtrl signs the admin out and returns them to the login page.
func adminLogoutCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := api.Http().Auth().SignOut(w); err != nil {
			api.Http().Response().Error(w, r, errors.New(api.Translate("error", "Unable to sign out")), http.StatusInternalServerError)
			return
		}
		api.Http().Response().FlashMsg(w, r, api.Translate("success", "Logged out successfully"), sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}
