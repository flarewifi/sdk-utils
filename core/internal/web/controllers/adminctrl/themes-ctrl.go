package adminctrl

import (
	"net/http"
	sdkapi "sdk/api"

	"core/internal/api"
	"core/resources/views/admin/themes"
	"tools/config"
)

func GetAvailableThemes(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		allPlugins := g.PluginMgr.All()
		adminThemes := []themes.ThemeOption{}
		portalThemes := []themes.ThemeOption{}

		for _, p := range allPlugins {
			features := p.Features()
			for _, f := range features {
				if f == "theme:admin" {
					info := p.Info()
					adminThemes = append(adminThemes, themes.ThemeOption{Label: info.Name, Value: info.Package})
				}
				if f == "theme:portal" {
					info := p.Info()
					portalThemes = append(portalThemes, themes.ThemeOption{Label: info.Name, Value: info.Package})
				}
			}
		}

		cfg, err := config.ReadThemesConfig()
		currentPortal := ""
		currentAdmin := ""
		if err == nil {
			currentPortal = cfg.PortalThemePkg
			currentAdmin = cfg.AdminThemePkg
		}

		page := themes.AdminThemesIndex(g.CoreAPI, portalThemes, adminThemes, currentPortal, currentAdmin)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})
	}
}

func SaveThemeSettings(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		formValidator := sdkapi.FormWithValidator{
			FormName: "themes",
			FormValidators: []sdkapi.FormValidator{
				{
					FieldName:  "portal_theme",
					FieldLabel: g.CoreAPI.Translate("label", "Select Portal Theme"),
					FieldType:  sdkapi.FormFieldTypeString,
					FieldRules: sdkapi.FormFieldRules{
						Required: true,
					},
				},
				{
					FieldName:  "admin_theme",
					FieldLabel: g.CoreAPI.Translate("label", "Select Admin Theme"),
					FieldType:  sdkapi.FormFieldTypeString,
					FieldRules: sdkapi.FormFieldRules{
						Required: true,
					},
				},
			},
		}

		err := g.CoreAPI.HttpAPI.Forms().ParseFormWithValidator(w, r, formValidator)
		if err != nil {
			themesIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:themes:index")
			http.Redirect(w, r, themesIndexUrl, http.StatusSeeOther)
			return
		}

		portalTheme := r.FormValue("portal_theme")
		adminTheme := r.FormValue("admin_theme")

		err = config.WriteThemesConfig(config.ThemesConfig{
			AdminThemePkg:  adminTheme,
			PortalThemePkg: portalTheme,
		})
		if err != nil {
			saveErrorMsg := g.CoreAPI.Translate("error", "save_settings_error")
			res.FlashMsg(w, r, saveErrorMsg, sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			themesIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:themes:index")
			http.Redirect(w, r, themesIndexUrl, http.StatusSeeOther)
			return
		}

		successfulSavedMsg := g.CoreAPI.Translate("info", "saved_settings_message")
		res.FlashMsg(w, r, successfulSavedMsg, sdkapi.FlashMsgSuccess)

		themesIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:themes:index")
		http.Redirect(w, r, themesIndexUrl, http.StatusSeeOther)
	}
}
