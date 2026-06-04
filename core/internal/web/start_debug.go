//go:build dev

package web

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"core/utils/env"

	"github.com/gorilla/mux"
)

func StartServer(r *mux.Router, forever bool) *http.Server {

	r.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)

	printRoutes(r)

	port := fmt.Sprintf(":%d", env.HTTP_PORT)

	srv := &http.Server{
		Handler: r,
		Addr:    port,
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
