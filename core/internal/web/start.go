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

	fmt.Println("+--------------------------------------------------------------+")
	fmt.Printf("| %-60s |\n", fmt.Sprintf("Listening on port %s", addr))
	fmt.Println("+--------------------------------------------------------------+")

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
