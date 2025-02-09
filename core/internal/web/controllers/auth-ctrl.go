package controllers

import (
	"core/internal/api"
	webutil "core/internal/utils/web"
	"net/http"
	sdkapi "sdk/api"
)

func AdminLoginCtrl(g *api.CoreGlobals) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := g.CoreAPI.HttpAPI.Auth().IsAuthenticated(r); err == nil {
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

		data := sdkapi.LoginPageData{
			LoginUrl: authUrl.String(),
		}

		page := t.PortalTheme.LoginPageFactory(w, r, data)
		g.CoreAPI.HttpAPI.HttpResponse().PortalView(w, r, page)
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
		g.CoreAPI.HttpAPI.HttpResponse().FlashMsg(w, r, "Logged in successfully", sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})
}
