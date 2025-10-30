//go:build dev

package main

import (
	"tools/cmd/livereload/dev"

	"log"
	"net/http"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	// LiveReload setup
	reloader := dev.NewLiveReloader()
	r.HandleFunc("/livereload", reloader.HandleWS)

	// Start watching files (async)
	go reloader.WatchPaths([]string{sdkutils.PathServerUp})

	log.Println("🚀 Live Reload started at http://0.0.0.0:8000")
	http.ListenAndServe(":8000", r)
}
