//go:build mono

package routes

import (
	"core/internal/api"
)

func AdminPluginRoutes(g *api.CoreGlobals) {
	// No plugin routes in monolithic build
}
