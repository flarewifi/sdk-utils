package coreforms

import (
	"core/internal/config"
	"core/internal/plugins"
	"net/http"
	sdkapi "sdk/api"
)

const (
	ThemesFormName = "themes"
)

func RegisterThemesForm(g *plugins.CoreGlobals) (err error) {
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
		Label: "Select Portal Theme",
		Type:  sdkapi.FormFieldTypeString,
		ValueFn: func() interface{} {
			cfg, err := config.ReadThemesConfig()
			if err != nil {
				return ""
			}
			return cfg.PortalThemePkg
		},
		Options: func() []sdkapi.FormListOption {
			opts := []sdkapi.FormListOption{}
			for _, p := range portalThemes {
				info := p.Info()
				opts = append(opts, sdkapi.FormListOption{
					Label: info.Name,
					Value: info.Package,
				})
			}
			return opts
		},
	}

	adminThemesField := sdkapi.FormListField{
		Name:  "admin_theme",
		Label: "Select Admin Theme",
		Type:  sdkapi.FormFieldTypeString,
		ValueFn: func() interface{} {
			cfg, err := config.ReadThemesConfig()
			if err != nil {
				return ""
			}
			return cfg.AdminThemePkg
		},
		Options: func() []sdkapi.FormListOption {
			opts := []sdkapi.FormListOption{}
			for _, p := range adminThemes {
				info := p.Info()
				opts = append(opts, sdkapi.FormListOption{
					Label: info.Name,
					Value: info.Package,
				})
			}
			return opts
		},
	}

	textField := sdkapi.FormTextField{
		Name:  "text_field",
		Label: "Text Field",
		ValueFn: func() string {
			val, err := g.CoreAPI.ConfigAPI.Plugin().Read("text_field")
			if err != nil {
				return err.Error()
			}
			return string(val)
		},
	}

	themesForm := sdkapi.HttpForm{
		CallbackRoute: "admin:themes:save",
		SubmitLabel:   "Save Settings",
		Sections: []sdkapi.FormSection{
			{
				Name: "themes",
				Fields: []sdkapi.IFormField{
					portalThemesField,
					adminThemesField,
					textField,
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
