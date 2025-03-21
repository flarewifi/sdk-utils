//go:build !dev

package web

import "github.com/gorilla/mux"

func printRoutes(router *mux.Router) {}
