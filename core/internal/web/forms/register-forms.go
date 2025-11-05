package coreforms

import (
	"core/internal/api"
)

func RegisterForms(g *api.CoreGlobals) {
	if err := RegisterThemesForm(g); err != nil {
		panic(err)
	}

	if err := RegisterLogsForm(g); err != nil {
		panic(err)
	}
}
