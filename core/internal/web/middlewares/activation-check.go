package middlewares

import (
	"net/http"
	"strings"

	"core/internal/modules/activation"
	"core/internal/web/helpers"
)

const (
	activationURL = "/activation"
	// bootStatusURL is the booting page's readiness probe. It must stay
	// activation-agnostic — see the exemption in ActivationCheck.
	bootStatusURL = "/boot/status"
)

// ActivationCheck ensures the device is activated before allowing access to app routes.
// Redirects to activation page if not activated.
// Allows access to: /activation, /activation/check, /activation/status and asset routes.
// NOTE: This middleware is applied in SetupAppRoutes(), so it only runs AFTER booting completes.
func ActivationCheck() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The boot readiness probe must answer truthfully regardless of
			// activation. The booting page polls it via XHR and only then does a
			// top-level navigation to "/". If activation gating turned this into a
			// redirect, the XHR would chase a cross-origin HTTPS hop (ForceHTTPS on
			// /activation), fail CORS with status 0, and the booting page would
			// never leave. Let the subsequent navigation handle activation routing.
			if r.URL.Path == bootStatusURL {
				next.ServeHTTP(w, r)
				return
			}

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
