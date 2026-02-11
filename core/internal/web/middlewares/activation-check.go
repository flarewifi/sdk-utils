package middlewares

import (
	"net/http"
	"strings"

	"core/internal/modules/activation"
	"core/internal/web/helpers"
)

const (
	activationURL = "/activation"
)

// ActivationCheck ensures the device is activated before allowing access to app routes.
// Redirects to activation page if not activated.
// Allows access to: /activation, /activation/check, /activation/status and asset routes.
// NOTE: This middleware is applied in SetupAppRoutes(), so it only runs AFTER booting completes.
func ActivationCheck() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isActivationRoute := r.URL.Path == activationURL || strings.HasPrefix(r.URL.Path, activationURL+"/")
			isActivationStatusRoute := r.URL.Path == activationURL+"/status"
			isActivated := activation.IsActivated.Load()

			// If already activated, redirect away from activation page
			// Exception: /activation/status always returns JSON for polling
			if isActivationRoute && isActivated && !isActivationStatusRoute {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}

			// Check if device is activated
			if isActivated {
				next.ServeHTTP(w, r)
				return
			}

			// Allow activation page and check endpoint
			if isActivationRoute {
				next.ServeHTTP(w, r)
				return
			}

			// Allow asset routes (JS, CSS, images, fonts)
			if helpers.IsAssetPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Not activated and not an allowed route - redirect to activation page
			http.Redirect(w, r, activationURL, http.StatusSeeOther)
		})
	}
}
