package adminctrl

import (
	"net/http"
	sdkapi "sdk/api"

	"core/internal/api"
	"core/resources/views/admin/themes"
	"core/utils/config"
)

// coreThemePkg is the built-in fallback theme. It is hidden from the picker —
// it is the implicit fallback when no real theme plugin is selected, not a
// user-selectable theme.
const coreThemePkg = "com.flarego.core"

// AdminThemesPage renders the admin dashboard theme picker (card grid).
func AdminThemesPage(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		cfg, _ := config.ReadThemesConfig()
		cards := buildThemeCards(g, "theme:admin", cfg.AdminThemePkg)
		saveUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:themes:save")
		page := themes.AdminThemesPage(g.CoreAPI, cards, cfg.AdminThemePkg, cfg.PortalThemePkg, saveUrl)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})
	}
}

// PortalThemesPage renders the captive portal theme picker (card grid).
func PortalThemesPage(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		cfg, _ := config.ReadThemesConfig()
		cards := buildThemeCards(g, "theme:portal", cfg.PortalThemePkg)
		saveUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:themes:save")
		page := themes.PortalThemesPage(g.CoreAPI, cards, cfg.PortalThemePkg, cfg.AdminThemePkg, saveUrl)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})
	}
}

// buildThemeCards collects the selectable themes for the given feature
// ("theme:admin" or "theme:portal"). Each theme's PreviewImage is resolved to
// a public URL scoped to that theme's own plugin (package + version), so the
// picker can show a sample. Themes without a PreviewImage get an empty URL and
// render a neutral placeholder in the view.
func buildThemeCards(g *api.CoreGlobals, feature string, currentPkg string) []themes.ThemeCard {
	cards := []themes.ThemeCard{}
	for _, p := range g.PluginMgr.Plugins() {
		info := p.Info()
		if info.Package == coreThemePkg {
			continue
		}
		for _, f := range p.Features() {
			if f != feature {
				continue
			}

			previewImg := ""
			if feature == "theme:admin" {
				previewImg = p.Themes().AdminPreviewImage()
			} else {
				previewImg = p.Themes().PortalPreviewImage()
			}

			previewURL := ""
			if previewImg != "" {
				previewURL = p.Http().Helpers().PublicPath(previewImg)
			}

			cards = append(cards, themes.ThemeCard{
				Package:     info.Package,
				Name:        info.Name,
				Description: info.Description,
				IsCurrent:   info.Package == currentPkg,
				PreviewURL:  previewURL,
			})
			break
		}
	}
	return cards
}

func SaveThemeSettings(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()

		formValidator := sdkapi.FormValidator{
			Name: "themes",
			Validators: []sdkapi.FormFieldValidator{
				{
					FieldName:  "portal_theme",
					FieldLabel: g.CoreAPI.Translate("label", "Select Portal Theme"),
					FieldType:  sdkapi.FormFieldTypeString,
					FieldRules: sdkapi.FormFieldRules{Required: true},
				},
				{
					FieldName:  "admin_theme",
					FieldLabel: g.CoreAPI.Translate("label", "Select Admin Theme"),
					FieldType:  sdkapi.FormFieldTypeString,
					FieldRules: sdkapi.FormFieldRules{Required: true},
				},
			},
		}

		formValues, err := g.CoreAPI.HttpAPI.Forms().ParseForm(w, r, formValidator)
		if err != nil {
			themesIndexUrl := themesRedirectURL(g, r)
			http.Redirect(w, r, themesIndexUrl, http.StatusSeeOther)
			return
		}

		portalTheme, _ := formValues.GetStringValue("portal_theme")
		adminTheme, _ := formValues.GetStringValue("admin_theme")

		err = config.WriteThemesConfig(config.ThemesConfig{
			AdminThemePkg:  adminTheme,
			PortalThemePkg: portalTheme,
		})
		if err != nil {
			saveErrorMsg := g.CoreAPI.Translate("error", "Unable to Save Settings")
			res.FlashMsg(w, r, saveErrorMsg, sdkapi.FlashMsgError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			themesIndexUrl := themesRedirectURL(g, r)
			http.Redirect(w, r, themesIndexUrl, http.StatusSeeOther)
			return
		}

		successfulSavedMsg := g.CoreAPI.Translate("info", "Settings Successfully Saved")
		res.FlashMsg(w, r, successfulSavedMsg, sdkapi.FlashMsgSuccess)

		themesIndexUrl := themesRedirectURL(g, r)
		http.Redirect(w, r, themesIndexUrl, http.StatusSeeOther)
	}
}

// themesRedirectURL returns the page to redirect back to after a theme save —
// the picker the request came from (admin or portal), falling back to the
// admin theme page when no referer is available.
func themesRedirectURL(g *api.CoreGlobals, r *http.Request) string {
	if ref := r.Referer(); ref != "" {
		return ref
	}
	return g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:themes:admin")
}
