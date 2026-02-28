//go:build !dev

package routes

import (
	"core/internal/api"
)

// WifiEventRoutes is a no-op in production.
// The fake WiFi event emitter is only available in dev mode.
func WifiEventRoutes(g *api.CoreGlobals) {
	// No-op in production
}
