//go:build !dev

package web

import (
	"fmt"
	"net/http"

	"core/utils/env"

	"github.com/gorilla/mux"
)

func StartServer(r *mux.Router, forever bool) *http.Server {
	addr := fmt.Sprintf(":%d", env.HTTP_PORT)

	fmt.Printf("Listening on port %s\n", addr)

	srv := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	if !forever {
		go func() {
			srv.ListenAndServe()
		}()
	} else {
		srv.ListenAndServe()
	}

	return srv
}
