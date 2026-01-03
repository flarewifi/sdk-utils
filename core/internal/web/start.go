//go:build !dev

package web

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"core/utils/env"

	"github.com/gorilla/mux"
)

func StartServer(r *mux.Router, forever bool) *http.Server {
	addr := fmt.Sprintf(":%d", env.HTTP_PORT)
	log.Println("Listening on port", addr)
	// log.Fatal(http.ListenAndServe(port, router.RootRouter()))

	srv := &http.Server{
		Handler: r,
		Addr:    addr,
		// Good practice: enforce timeouts for servers you create!
		// WriteTimeout: 15 * time.Second,
		// ReadTimeout:  15 * time.Second,
	}

	if !forever {
		go func() {
			err := srv.ListenAndServe()
			if err != nil && !errors.Is(http.ErrServerClosed, err) {
				log.Printf("Error starting server: %v\n", err)
			}
		}()
	} else {
		log.Fatal(srv.ListenAndServe())
	}

	return srv
}
