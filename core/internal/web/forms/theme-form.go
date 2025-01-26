package coreforms

import (
	"core/internal/config"
	"core/internal/plugins"
	"encoding/json"
	sdkapi "sdk/api"
)

const (
	ThemesFormName = "themes"
)

type MultiFieldRowData struct {
	Col1 string
	Col2 float64
	Col3 int64
	Col4 bool
}

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
		Type:  sdkapi.FormFieldTypeText,
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
		Type:  sdkapi.FormFieldTypeText,
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

	multiField := sdkapi.FormMultiField{
		Name:  "multi_field",
		Label: "Multi Field",
		Columns: func() []sdkapi.FormMultiFieldCol {
			cols := []sdkapi.FormMultiFieldCol{
				{
					Name:  "col1",
					Label: "Column 1 (text)",
					Type:  sdkapi.FormFieldTypeText,
					ValueFn: func() interface{} {
						return "text value"
					},
				},
				{
					Name:  "col2",
					Label: "Column 2 (decimal)",
					Type:  sdkapi.FormFieldTypeDecimal,
					ValueFn: func() interface{} {
						return 100.1
					},
				},
				{
					Name:  "col3",
					Label: "Column 3 (integer)",
					Type:  sdkapi.FormFieldTypeInteger,
					ValueFn: func() interface{} {
						return 1
					},
				},
				{
					Name:  "col4",
					Label: "Column 4 (boolean)",
					Type:  sdkapi.FormFieldTypeBoolean,
					ValueFn: func() interface{} {
						return true
					},
				},
			}
			return cols
		},
		ValueFn: func() [][]sdkapi.FormFieldData {
			var rowData []MultiFieldRowData
			b, err := g.CoreAPI.Config().Plugin().Read("multi_field")
			if err != nil {
				return nil
			}

			err = json.Unmarshal(b, &rowData)
			if err != nil {
				return nil
			}

			data := make([][]sdkapi.FormFieldData, len(rowData))
			for i, row := range rowData {
				data[i] = []sdkapi.FormFieldData{
					{
						Name:  "col1",
						Value: row.Col1,
					},
					{
						Name:  "col2",
						Value: row.Col2,
					},
					{
						Name:  "col3",
						Value: row.Col3,
					},
					{
						Name:  "col4",
						Value: row.Col4,
					},
				}
			}

			return data
		},
	}

	listFieldTxt := sdkapi.FormListField{
		Name:     "list_field_txt",
		Label:    "List Field (text)",
		Multiple: true,
		Type:     sdkapi.FormFieldTypeText,
		Options: func() []sdkapi.FormListOption {
			return []sdkapi.FormListOption{
				{
					Label: "Value 1",
					Value: "val1",
				},
				{
					Label: "Value 2",
					Value: "val2",
				},
			}
		},
		ValueFn: func() interface{} {
			return []string{"val1", "val2"}
		},
	}

	listFieldNum := sdkapi.FormListField{
		Name:     "list_field_num",
		Label:    "List Field (number)",
		Type:     sdkapi.FormFieldTypeDecimal,
		Multiple: true,
		Options: func() []sdkapi.FormListOption {
			return []sdkapi.FormListOption{
				{
					Label: "100",
					Value: 100.0,
				},
				{
					Label: "200",
					Value: 200.0,
				},
			}
		},
		ValueFn: func() interface{} {
			return []float64{100.0, 200.0}
		},
	}

	textField := sdkapi.FormTextField{
		Name:  "text_field",
		Label: "Text Field",
		ValueFn: func() string {
			return "text value"
		},
	}

	intField := sdkapi.FormIntegerField{
		Name:  "int_field",
		Label: "Int Field",
		ValueFn: func() int64 {
			return 124
		},
	}

	decimalField := sdkapi.FormDecimalField{
		Name:      "decimal_field",
		Label:     "Decimal Field",
		Step:      0.1,
		Precision: 2,
		ValueFn: func() float64 {
			return 201.50
		},
	}

	boolField := sdkapi.FormBooleanField{
		Name:  "boolean_field",
		Label: "Boolean Field",
		ValueFn: func() bool {
			return true
		},
	}

	themesForm := sdkapi.HttpForm{
		Name:          ThemesFormName,
		CallbackRoute: "admin:themes:save",
		SubmitLabel:   "Save",
		Sections: []sdkapi.FormSection{
			{
				Name: "themes",
				Fields: []sdkapi.IFormField{
					textField,
					intField,
					decimalField,
					boolField,
					portalThemesField,
					adminThemesField,
					multiField,
					listFieldTxt,
					listFieldNum,
				},
			},
		},
	}

	err = g.CoreAPI.HttpAPI.Forms().RegisterForms(themesForm)
	if err != nil {
		return err
	}

	return nil
}
