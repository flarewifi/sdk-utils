//go:build dev

package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func printRoutes(router *mux.Router) {
	router.PathPrefix("/routes").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<h1>Routes</h1><br><ol>")
		router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			pathTemplate, _ := route.GetPathTemplate()
			methods, _ := route.GetMethods()
			name := route.GetName()
			m := strings.Join(methods, ",")
			if pathTemplate != "" && m != "" {
				fmt.Fprintf(w, "<li>%s %s (%s)</li>", m, pathTemplate, name)
			}
			return nil
		})
		fmt.Fprint(w, "</ol>")
	})
}
