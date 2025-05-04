package coreforms

import (
	"core/internal/api"
	"net/http"
	sdkapi "sdk/api"
)

func RegisterLogsForm(g *api.CoreGlobals) error {
	return g.CoreAPI.HttpAPI.Forms().RegisterForm("logs-form", func(r *http.Request) sdkapi.HttpForm {

		params := r.URL.Query()
		pkg := params.Get("package")
		level := params.Get("level")
		searchText := params.Get("search_text")

		return sdkapi.HttpForm{
			CallbackRoute: "admin:logs:search",
			SubmitLabel:   g.CoreAPI.Translate("label", "system_logs"),
			Sections: []sdkapi.FormSection{
				{
					Name:  "search",
					Label: g.CoreAPI.Translate("label", "system_logs"),
					Fields: []sdkapi.IFormField{
						sdkapi.FormStringField{
							Name:     "search_text",
							Label:    g.CoreAPI.Translate("label", "search_logs"),
							Required: true,
							ValueFn: func() string {
								return searchText
							},
						},
						sdkapi.FormListField{
							Name:  "package",
							Label: g.CoreAPI.Translate("label", "package"),
							Type:  sdkapi.FormFieldTypeString,
							Options: func() []sdkapi.FormListFieldOption {
								opts := []sdkapi.FormListFieldOption{
									{Label: g.CoreAPI.Translate("label", "logs_all"), Value: ""},
								}

								pkgs := g.PluginMgr.All()
								for _, pkg := range pkgs {
									info := pkg.Info()
									opt := sdkapi.FormListFieldOption{
										Label: info.Package,
										Value: info.Package,
									}
									opts = append(opts, opt)
								}
								return opts
							},
							ValueFn: func() interface{} {
								return pkg
							},
						},
						sdkapi.FormListField{
							Name:       "level",
							Label:      g.CoreAPI.Translate("label", "level"),
							Type:       sdkapi.FormFieldTypeString,
							OptionType: sdkapi.OptionTypeSelect,
							Options: func() []sdkapi.FormListFieldOption {
								return []sdkapi.FormListFieldOption{
									{Label: g.CoreAPI.Translate("label", "logs_all"), Value: ""},
									{Label: g.CoreAPI.Translate("label", "logs_info"), Value: api.LogLevelInfo},
									{Label: g.CoreAPI.Translate("label", "logs_debug"), Value: api.LogLevelDebug},
									{Label: g.CoreAPI.Translate("label", "logs_error"), Value: api.LogLevelError},
								}
							},
							ValueFn: func() interface{} {
								return level
							},
						},
					},
				},
			},
		}

	})
}
