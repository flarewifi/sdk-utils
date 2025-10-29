package adminctrl

import (
	"errors"
	"net/http"
	sdkapi "sdk/api"

	"core/internal/api"
	coreforms "core/internal/web/forms"
	"core/resources/views/admin/themes"
	"tools/config"
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
		saveErrorMsg := errors.New(g.CoreAPI.Translate("error", "save_settings_error"))

		httpForm, err := g.CoreAPI.HttpAPI.Forms().ParseForm(coreforms.ThemesFormName, w, r)
		if err != nil {
			res.Error(w, r, saveErrorMsg, http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		portalTheme, err := httpForm.GetStringValue("themes", "portal_theme")
		if err != nil {
			res.Error(w, r, saveErrorMsg, http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		adminTheme, err := httpForm.GetStringValue("themes", "admin_theme")
		if err != nil {
			res.Error(w, r, saveErrorMsg, http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		err = config.WriteThemesConfig(config.ThemesConfig{
			AdminThemePkg:  adminTheme,
			PortalThemePkg: portalTheme,
		})
		if err != nil {
			res.Error(w, r, saveErrorMsg, http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		successfulSavedMsg := g.CoreAPI.Translate("info", "saved_settings_message")
		api.NewGlobals().CoreAPI.HttpAPI.Response().FlashMsg(w, r, successfulSavedMsg, sdkapi.FlashMsgSuccess)

		themesIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:themes:index")
		http.Redirect(w, r, themesIndexUrl, http.StatusSeeOther)
	}
}
