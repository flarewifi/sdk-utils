package adminctrl

import (
	"net/http"

	"core/internal/api"
	"core/internal/modules/logger"
)

type LogViewerData struct {
	Logs  []*logger.LogLine `json:"logs"`
	Count int               `json:"count"`
}

// Gets the logs based on the requested current page and
// per page queries
func GetLogs(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement log retrieval functionality
	}
}

func ClearLogs(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement log clearing functionality
	}
}
