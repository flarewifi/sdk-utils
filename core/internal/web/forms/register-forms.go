package coreforms

import (
	"core/internal/plugins"
)

func RegisterForms(g *plugins.CoreGlobals) {
	if err := RegisterThemesForm(g); err != nil {
		panic(err)
	}

	if err := RegisterLogsForm(g); err != nil {
		panic(err)
	}
}
