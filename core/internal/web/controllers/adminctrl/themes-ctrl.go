package adminctrl

import (
	"net/http"
	sdkhttp "sdk/api/http"

	"core/internal/config"
	"core/internal/plugins"
	coreforms "core/internal/web/forms"
	"core/resources/views/admin/themes"
)

func GetAvailableThemes(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.HttpResponse()

		themeForm, err := coreforms.GetThemeForm(g)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		err = g.CoreAPI.HttpAPI.Forms().RegisterHttpForms(themeForm)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		httpForm, err := g.CoreAPI.HttpAPI.Forms().GetForm(themeForm.Name)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		page := themes.AdminThemesIndex(httpForm.Template(r))
		res.AdminView(w, r, sdkhttp.ViewPage{PageContent: page})
	}
}

func SaveThemeSettings(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.HttpResponse()
		f, err := coreforms.GetThemeForm(g)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		httpForm, err := g.CoreAPI.HttpAPI.Forms().GetForm(f.Name)
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

		themesIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:themes:index")
		http.Redirect(w, r, themesIndexUrl, http.StatusSeeOther)
	}
}
