package controllers

import (
	"core/internal/plugins"
	webutil "core/internal/utils/web"
	"net/http"
	sdkhttp "sdk/api/http"
)

func AdminLoginCtrl(g *plugins.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.CoreAPI.HttpAPI.Auth().IsAuthenticated(r) {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		res := g.CoreAPI.HttpAPI.HttpResponse()
		_, t, err := g.PluginMgr.GetAdminTheme()
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		authRoute := webutil.RootRouter.Get("admin:authenticate")
		authUrl, _ := authRoute.URL()

		data := sdkhttp.LoginPageData{
			LoginUrl: authUrl.String(),
		}

		page := t.PortalTheme.LoginPageFactory(w, r, data)
		g.CoreAPI.HttpAPI.HttpResponse().PortalView(w, r, page)
	})
}

func AdminAuthenticateCtrl(g *plugins.CoreGlobals) http.Handler {
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
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})
}
