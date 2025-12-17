package middlewares

import (
	"net/http"

	sdkapi "sdk/api"
)

// EnsureDeviceRegistered middleware checks if client device is registered
// This middleware should run before any plugin-registered portal middlewares
func EnsureDeviceRegistered(api sdkapi.IPluginApi) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client device is registered before showing portal index
			clnt, err := api.Http().GetClientDevice(r)
			if err != nil || clnt == nil {
				// Device not found - redirect to registration
				api.Http().Response().RedirectToPortal(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
