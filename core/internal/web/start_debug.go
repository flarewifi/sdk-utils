//go:build dev

package web

import (
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"core/internal/modules/logger"
	"core/utils/env"

	"github.com/gorilla/mux"
)

// StartServer starts the HTTP server on a background goroutine and returns
// its *http.Server handle immediately, so the caller can later call
// Shutdown(ctx) on it for a graceful stop.
func StartServer(r *mux.Router) *http.Server {

	r.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)

	printRoutes(r)

	port := fmt.Sprintf(":%d", env.HTTP_PORT)

	fmt.Println("+--------------------------------------------------------------+")
	fmt.Printf("| %-60s |\n", fmt.Sprintf("Listening on port %s", port))
	fmt.Println("+--------------------------------------------------------------+")

	srv := &http.Server{
		Handler: r,
		Addr:    port,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			file, line := logger.GetCallerFileLine(0)
			logger.Emit(2, file, line, fmt.Sprintf("http server on %s stopped: %v", port, err))
		}
	}()

	return srv
}
