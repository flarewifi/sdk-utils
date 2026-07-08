//go:build !dev

package web

import (
	"errors"
	"fmt"
	"net/http"

	"core/internal/modules/logger"
	"core/utils/env"

	"github.com/gorilla/mux"
)

// StartServer starts the HTTP server on a background goroutine and returns
// its *http.Server handle immediately, so the caller can later call
// Shutdown(ctx) on it for a graceful stop.
func StartServer(r *mux.Router) *http.Server {
	addr := fmt.Sprintf(":%d", env.HTTP_PORT)

	fmt.Println("+--------------------------------------------------------------+")
	fmt.Printf("| %-60s |\n", fmt.Sprintf("Listening on port %s", addr))
	fmt.Println("+--------------------------------------------------------------+")

	srv := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			file, line := logger.GetCallerFileLine(0)
			logger.Emit(2, file, line, fmt.Sprintf("http server on %s stopped: %v", addr, err))
		}
	}()

	return srv
}
