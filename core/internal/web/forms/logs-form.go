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
			SubmitLabel:   "Search Logs",
			Sections: []sdkapi.FormSection{
				{
					Name:  "search",
					Label: "System Logs",
					Fields: []sdkapi.IFormField{
						sdkapi.FormFileField{
							Name:      "upload_file",
							Label:     "Upload File",
							Required:  true,
							Multiple:  true,
							MinFiles:  1,
							MaxFiles:  3,
							MinSizeMb: 1, // 1mb
							MaxSizeMb: 10,
							Accept:    []string{"application/zip", "image/png", "image/jpeg"},
							ValueFn: func() []string {
								return nil
							},
						},
						sdkapi.FormStringField{
							Name:     "search_text",
							Label:    "Search Logs",
							Required: true,
							ValueFn: func() string {
								return searchText
							},
						},
						sdkapi.FormListField{
							Name:  "package",
							Label: "Package",
							Type:  sdkapi.FormFieldTypeString,
							Options: func() []sdkapi.FormListFieldOption {
								opts := []sdkapi.FormListFieldOption{
									{Label: "All", Value: ""},
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
							Name:  "level",
							Label: "Level",
							Type:  sdkapi.FormFieldTypeString,
							Options: func() []sdkapi.FormListFieldOption {
								return []sdkapi.FormListFieldOption{
									{Label: "All", Value: ""},
									{Label: "Info", Value: api.LogLevelInfo},
									{Label: "Debug", Value: api.LogLevelDebug},
									{Label: "Error", Value: api.LogLevelError},
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
