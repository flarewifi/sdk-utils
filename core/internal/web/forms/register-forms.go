package coreforms

import (
	"core/internal/api"
)

func RegisterForms(g *api.CoreGlobals) {
	if err := RegisterLogsForm(g); err != nil {
		panic(err)
	}
}
