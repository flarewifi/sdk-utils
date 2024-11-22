package coreforms

import (
	"core/internal/plugins"
)

func RegisterForms(g *plugins.CoreGlobals) {

	themesForm, err := GetThemeForm(g)
	if err != nil {
		panic(err)
	}

	g.CoreAPI.HttpAPI.Forms().RegisterHttpForms(themesForm)
}
