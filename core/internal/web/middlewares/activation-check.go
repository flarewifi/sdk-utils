package middlewares

import (
	"net/http"
	"strings"

	"core/internal/utils/activation"
	"core/internal/web/helpers"
)

const (
	activationURL = "/activation"
)

// ActivationCheck ensures the device is activated before allowing access to app routes.
// Redirects to activation page if not activated.
// Allows access to: /activation, /activation/check and asset routes.
// NOTE: This middleware is applied in SetupAppRoutes(), so it only runs AFTER booting completes.
func ActivationCheck() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if device is activated
			if activation.IsActivated.Load() {
				next.ServeHTTP(w, r)
				return
			}

			// Allow activation page and check endpoint
			if r.URL.Path == activationURL || strings.HasPrefix(r.URL.Path, activationURL+"/") {
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
