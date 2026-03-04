//go:build dev

package httpsserver

import (
	"github.com/gorilla/mux"
)

// StartHTTPSServer is disabled in dev mode - only HTTP on port 3000 is used
func StartHTTPSServer(r *mux.Router) error {
	// No-op in dev mode - HTTPS server is disabled
	return nil
}

// StopHTTPSServer is disabled in dev mode
func StopHTTPSServer() {
	// No-op in dev mode
}

// IsHTTPSServerRunning always returns false in dev mode
func IsHTTPSServerRunning() bool {
	return false
}

// GetCurrentRouter returns nil in dev mode
func GetCurrentRouter() *mux.Router {
	return nil
}
