package controllers

import (
	"core/internal/api"
	"core/internal/web/middlewares"
	"errors"
	"net/http"
	sdkapi "sdk/api"
)

func AdminLoginCtrl(g *api.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Render the login page over HTTPS. Its form posts to the HTTPS-only auth
		// route, so an HTTP page would post over HTTP and get bounced HTTP→HTTPS
		// (302) — which browsers downgrade to a GET, dropping the credentials and
		// kicking the user back to the form. In prod ForceHTTPS already serves
		// /login over HTTPS (this is a no-op); in dev (plain HTTP) this redirect is
		// what puts the page — and therefore its form POST — on HTTPS.
		if !middlewares.IsHTTPS(r) {
			http.Redirect(w, r, middlewares.HTTPSURL(r), http.StatusSeeOther)
			return
		}

		if _, err := g.CoreAPI.HttpAPI.Auth().IsAuthenticated(r); err == nil {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		res := g.CoreAPI.HttpAPI.Response()
		p, t, _, err := g.PluginMgr.GetPortalTheme()
		if err != nil {
			res.Error(w, r, errors.New(g.CoreAPI.Translate("error", "Unable to Get Admin Theme")), http.StatusInternalServerError)
			return
		}

		// Check for flash error message
		var loginErr error
		flashType, _ := g.CoreAPI.HttpAPI.Cookie().GetCookie(r, "flash_type")
		flashMsg, _ := g.CoreAPI.HttpAPI.Cookie().GetCookie(r, "flash_message")
		if flashType == sdkapi.FlashMsgError && flashMsg != "" {
			loginErr = errors.New(flashMsg)
			// Clear flash cookies after reading
			g.CoreAPI.HttpAPI.Cookie().DeleteCookie(w, "flash_type")
			g.CoreAPI.HttpAPI.Cookie().DeleteCookie(w, "flash_message")
		}

		data := sdkapi.LoginPageData{
			LoginError:        loginErr,
			ForgotPasswordUrl: g.CoreAPI.HttpAPI.Helpers().UrlForRoute("auth:send-otp"),
		}

		page := t.PortalTheme.LoginPageFactory(w, r, data)
		p.Http().Response().PortalView(w, r, page)
	})
}

// AdminAuthenticateCtrl handles POST /login for the fallback (core) login form.
// When the configured theme plugin is not loaded, this endpoint processes credentials.
func AdminAuthenticateCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			g.CoreAPI.HttpAPI.Response().FlashMsg(w, r, g.CoreAPI.Translate("error", "Invalid form data"), sdkapi.FlashMsgError)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		acct, err := g.CoreAPI.HttpAPI.Auth().Authenticate(username, password)
		if err != nil {
			g.CoreAPI.HttpAPI.Response().FlashMsg(w, r, g.CoreAPI.Translate("error", "Invalid credentials"), sdkapi.FlashMsgError)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		g.CoreAPI.HttpAPI.Auth().SignIn(w, acct)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func AdminLogoutCtrl(g *api.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := g.CoreAPI.HttpAPI.Auth().SignOut(w); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		g.CoreAPI.HttpAPI.Response().FlashMsg(w, r, g.CoreAPI.Translate("success", "Logged out successfully"), sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
}
