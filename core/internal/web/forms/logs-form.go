package coreforms

import (
	"core/internal/plugins"
	"net/http"
	sdkapi "sdk/api"
)

func RegisterLogsForm(g *plugins.CoreGlobals) error {
	return g.CoreAPI.HttpAPI.Forms().RegisterForm("logs-form", func(r *http.Request) sdkapi.HttpForm {

		params := r.URL.Query()
		pkg := params.Get("package")
		level := params.Get("level")
		searchText := params.Get("search_text")

		return sdkapi.HttpForm{
			CallbackRoute: "admin:logs:search",
			SubmitLabel:   "Search Logs",
			Sections: []sdkapi.FormSection{
				{
					Name:  "search",
					Label: "System Logs",
					Fields: []sdkapi.IFormField{
						sdkapi.FormTextField{
							Name:  "search_text",
							Label: "Search Logs",
							ValueFn: func() string {
								return searchText
							},
						},
						sdkapi.FormListField{
							Name:  "package",
							Label: "Package",
							Type:  sdkapi.FormFieldTypeText,
							Options: func() []sdkapi.FormListOption {
								opts := []sdkapi.FormListOption{
									{Label: "All", Value: ""},
								}

								pkgs := g.PluginMgr.All()
								for _, pkg := range pkgs {
									info := pkg.Info()
									opt := sdkapi.FormListOption{
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
							Name:  "level",
							Label: "Level",
							Type:  sdkapi.FormFieldTypeText,
							Options: func() []sdkapi.FormListOption {
								return []sdkapi.FormListOption{
									{Label: "All", Value: ""},
									{Label: "Info", Value: plugins.LogLevelInfo},
									{Label: "Debug", Value: plugins.LogLevelDebug},
									{Label: "Error", Value: plugins.LogLevelError},
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
