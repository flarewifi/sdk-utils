package adminctrl

import (
	"net/http"
	sdkapi "sdk/api"

	"core/internal/api"
	"core/internal/config"
	coreforms "core/internal/web/forms"
	"core/resources/views/admin/themes"
)

func GetAvailableThemes(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		formTpl, err := g.CoreAPI.HttpAPI.Forms().GetFormTemplate(coreforms.ThemesFormName, r)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		page := themes.AdminThemesIndex(formTpl)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})
	}
}

func SaveThemeSettings(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		httpForm, err := g.CoreAPI.HttpAPI.Forms().ParseForm(coreforms.ThemesFormName, w, r)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		portalTheme, err := httpForm.GetStringValue("themes", "portal_theme")
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		adminTheme, err := httpForm.GetStringValue("themes", "admin_theme")
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		err = config.WriteThemesConfig(config.ThemesConfig{
			AdminThemePkg:  adminTheme,
			PortalThemePkg: portalTheme,
		})
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		api.NewGlobals().CoreAPI.HttpAPI.Response().FlashMsg(w, r, "Settings saved successfully", sdkapi.FlashMsgSuccess)
		themesIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:themes:index")
		http.Redirect(w, r, themesIndexUrl, http.StatusSeeOther)
	}
}
