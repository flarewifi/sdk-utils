package controllers

import (
	"core/internal/api"
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
			LoginError: loginErr,
		}

		page := t.PortalTheme.LoginPageFactory(w, r, data)
		p.Http().Response().PortalView(w, r, page)
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
