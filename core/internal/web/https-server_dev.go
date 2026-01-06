//go:build dev

package web

import (
	"github.com/gorilla/mux"
)

// StartHTTPSServer is disabled in dev mode - only HTTP on port 3000 is used
func StartHTTPSServer(r *mux.Router) {
	// No-op in dev mode - HTTPS server is disabled
}
