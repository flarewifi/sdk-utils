package adminctrl

import (
	"encoding/json"
	"fmt"
	"net/http"
	sdkapi "sdk/api"

	"core/internal/config"
	"core/internal/plugins"
	coreforms "core/internal/web/forms"
	"core/resources/views/admin/themes"
)

func GetAvailableThemes(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.HttpResponse()
		formTpl, err := g.CoreAPI.HttpAPI.Forms().GetFormTemplate(coreforms.ThemesFormName, r)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		page := themes.AdminThemesIndex(formTpl)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})
	}
}

func SaveThemeSettings(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.HttpResponse()
		httpForm, err := g.CoreAPI.HttpAPI.Forms().ParseForm(coreforms.ThemesFormName, r)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		mfdata, err := httpForm.GetMultiField("themes", "multi_field")
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		fmt.Printf("mfdata: %+v\n", mfdata)

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

		// extras for testing
		mfd, err := httpForm.GetMultiField("themes", "multi_field")
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		numRows := mfd.NumRows()
		data := make([]coreforms.MultiFieldRowData, numRows)
		for i := 0; i < numRows; i++ {
			var row coreforms.MultiFieldRowData
			col1, err := mfd.GetStringValue(i, "col1")
			if err != nil {
				res.Error(w, r, err, http.StatusInternalServerError)
				return
			}
			row.Col1 = col1

			col2, err := mfd.GetFloatValue(i, "col2")
			if err != nil {
				res.Error(w, r, err, http.StatusInternalServerError)
				return
			}
			row.Col2 = col2

			col3, err := mfd.GetIntValue(i, "col3")
			if err != nil {
				res.Error(w, r, err, http.StatusInternalServerError)
				return
			}
			row.Col3 = col3

			col4, err := mfd.GetBoolValue(i, "col4")
			if err != nil {
				res.Error(w, r, err, http.StatusInternalServerError)
				return
			}
			row.Col4 = col4
			data[i] = row
		}

		b, err := json.Marshal(data)
		if err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		if err := g.CoreAPI.Config().Plugin().Write("multi_field", b); err != nil {
			res.Error(w, r, err, http.StatusInternalServerError)
			return
		}

		themesIndexUrl := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:themes:index")
		http.Redirect(w, r, themesIndexUrl, http.StatusSeeOther)
	}
}
