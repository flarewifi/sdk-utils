package web

import (
	"core/internal/web/httpsserver"

	"github.com/gorilla/mux"
)

// StartHTTPSServer starts the HTTPS server if enabled in config
// This is a wrapper that delegates to the httpsserver package
func StartHTTPSServer(r *mux.Router) error {
	return httpsserver.StartHTTPSServer(r)
}

// StopHTTPSServer gracefully stops the HTTPS server
func StopHTTPSServer() {
	httpsserver.StopHTTPSServer()
}

// IsHTTPSServerRunning returns true if the HTTPS server is currently running
func IsHTTPSServerRunning() bool {
	return httpsserver.IsHTTPSServerRunning()
}
