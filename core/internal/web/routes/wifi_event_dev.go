//go:build dev

package routes

import (
	"encoding/json"
	"net/http"
	"strings"

	"core/internal/api"
	"core/internal/web/router"
	sdkapi "sdk/api"

	"github.com/gorilla/mux"
)

// WifiEventRoutes registers dev-only route for emitting fake WiFi events.
// This allows testing auto-pause and other WiFi event handlers without real hardware.
//
// Usage:
//
//	GET /wifi-event/{interface}/{event}/{mac}
//
// Examples:
//
//	GET /wifi-event/wlan0/connected/AA:BB:CC:DD:EE:FF
//	GET /wifi-event/wlan0/disconnected/AA:BB:CC:DD:EE:FF
func WifiEventRoutes(g *api.CoreGlobals) {
	router.RootRouter.HandleFunc("/wifi-event/{interface}/{event}/{mac}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		iface := vars["interface"]
		eventStr := strings.ToLower(vars["event"])
		mac := strings.ToUpper(vars["mac"])

		// Map event string to WifiClientEvent
		var event sdkapi.WifiClientEvent
		switch eventStr {
		case "connected", "connect":
			event = sdkapi.WifiEventClientConnected
		case "disconnected", "disconnect":
			event = sdkapi.WifiEventClientDisconnected
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "invalid event type",
				"message": "event must be 'connected' or 'disconnected'",
			})
			return
		}

		// Emit the WiFi event to all registered handlers
		api.EmitWifiEvent(event, mac)

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "ok",
			"interface": iface,
			"event":     string(event),
			"mac":       mac,
		})
	}).Methods(http.MethodGet).Name("dev:wifi-event")
}
