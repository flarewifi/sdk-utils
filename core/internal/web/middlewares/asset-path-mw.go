package middlewares

import (
	"core/internal/web/helpers"
	"net/http"
)

func AssetPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if helpers.IsAssetPath(r.URL.Path) {
			next.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 page not found"))
		}
	})
}
