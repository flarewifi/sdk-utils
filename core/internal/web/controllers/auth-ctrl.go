package controllers

import (
	"core/internal/api"
	webutil "core/internal/utils/web"
	"errors"
	"net/http"
	sdkapi "sdk/api"
)

func AdminLoginCtrl(g *api.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := g.CoreAPI.HttpAPI.Auth().IsAuthenticated(r); err == nil {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		res := g.CoreAPI.HttpAPI.Response()
		p, t, err := g.PluginMgr.GetPortalTheme()
		if err != nil {
			res.Error(w, r, errors.New(g.CoreAPI.Translate("error", "get_admin_theme_error")), http.StatusInternalServerError)
			return
		}

		authRoute := webutil.RootRouter.Get("admin:authenticate")
		authUrl, _ := authRoute.URL()

		data := sdkapi.LoginPageData{
			LoginUrl: authUrl.String(),
		}

		page := t.PortalTheme.LoginPageFactory(w, r, data)
		p.Http().Response().PortalView(w, r, page)
	})
}

func AdminAuthenticateCtrl(g *api.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			// TODO: Handle error
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		acct, err := g.CoreAPI.HttpAPI.Auth().Authenticate(username, password)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		g.CoreAPI.HttpAPI.Auth().SignIn(w, acct)
		g.CoreAPI.HttpAPI.Response().FlashMsg(w, r, "Logged in successfully", sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})
}

func AdminLogoutCtrl(g *api.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := g.CoreAPI.HttpAPI.Auth().SignOut(w); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		g.CoreAPI.HttpAPI.Response().FlashMsg(w, r, "Logged out successfully", sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
}
