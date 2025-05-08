package coreforms

import (
	"core/internal/api"
	"core/internal/config"
	"net/http"
	sdkapi "sdk/api"
)

const (
	ThemesFormName = "themes"
)

func RegisterThemesForm(g *api.CoreGlobals) (err error) {
	allPlugins := g.PluginMgr.All()
	adminThemes := []sdkapi.IPluginApi{}
	portalThemes := []sdkapi.IPluginApi{}

	for _, p := range allPlugins {
		features := p.Features()
		for _, f := range features {
			if f == "theme:admin" {
				adminThemes = append(adminThemes, p)
			}

			if f == "theme:portal" {
				portalThemes = append(portalThemes, p)
			}
		}
	}

	portalThemesField := sdkapi.FormListField{
		Name:  "portal_theme",
		Label: g.CoreAPI.Translate("label", "select_portal_theme"),
		Type:  sdkapi.FormFieldTypeString,
		ValueFn: func() interface{} {
			cfg, err := config.ReadThemesConfig()
			if err != nil {
				return ""
			}
			return cfg.PortalThemePkg
		},
		Options: func() []sdkapi.FormListFieldOption {
			opts := []sdkapi.FormListFieldOption{}
			for _, p := range portalThemes {
				info := p.Info()
				opts = append(opts, sdkapi.FormListFieldOption{
					Label: info.Name,
					Value: info.Package,
				})
			}
			return opts
		},
	}

	adminThemesField := sdkapi.FormListField{
		Name:  "admin_theme",
		Label: g.CoreAPI.Translate("label", "select_admin_theme"),
		Type:  sdkapi.FormFieldTypeString,
		ValueFn: func() interface{} {
			cfg, err := config.ReadThemesConfig()
			if err != nil {
				return ""
			}
			return cfg.AdminThemePkg
		},
		Options: func() []sdkapi.FormListFieldOption {
			opts := []sdkapi.FormListFieldOption{}
			for _, p := range adminThemes {
				info := p.Info()
				opts = append(opts, sdkapi.FormListFieldOption{
					Label: info.Name,
					Value: info.Package,
				})
			}
			return opts
		},
	}

	themesForm := sdkapi.HttpForm{
		CallbackRoute: "admin:themes:save",
		SubmitLabel:   g.CoreAPI.Translate("label", "save_settings"),
		Sections: []sdkapi.FormSection{
			{
				Name: "themes",
				Fields: []sdkapi.IFormField{
					portalThemesField,
					adminThemesField,
				},
			},
		},
	}

	err = g.CoreAPI.HttpAPI.Forms().RegisterForm(ThemesFormName, func(r *http.Request) sdkapi.HttpForm {
		return themesForm
	})

	if err != nil {
		return err
	}

	return nil
}
