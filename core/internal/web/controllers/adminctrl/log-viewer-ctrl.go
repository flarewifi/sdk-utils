package adminctrl

import (
	"net/http"

	"core/internal/plugins"
	"core/internal/utils/logger"
)

type LogViewerData struct {
	Logs  []*logger.LogLine `json:"logs"`
	Count int               `json:"count"`
}

// Gets the logs based on the requested current page and
// per page queries
func GetLogs(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()

		// // test
		// rows := int(logger.LineCount.Load())
		// g.CoreAPI.LoggerAPI.Debug("test "+fmt.Sprintf("%d", rows), "test body")

		// currentPage := sdkstr.AtoiOrDefault(r.URL.Query().Get("currentPage"), 1)
		// perPage := sdkstr.AtoiOrDefault(r.URL.Query().Get("perPage"), 50)
		// count := int(logger.LineCount.Load())

		// // set start and end lines based on the
		// // currentPage and perPage query
		// start := (perPage * (currentPage - 1))
		// if start < 0 {
		// 	start = 0
		// }

		// end := start + perPage - 1
		// if end > count {
		// 	end = count
		// }

		// // read logs
		// logs, err := logger.ReadLogs(start, end)
		// if err != nil {
		// 	log.Println(err)
		// 	res.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// data := LogViewerData{
		// 	Logs:  logs,
		// 	Count: count,
		// }

		// res.Json(w, data, http.StatusOK)
	}
}

func ClearLogs(g *plugins.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// res := g.CoreAPI.HttpAPI.VueResponse()
		// err := logger.ClearLogs()
		// if err != nil {
		// 	log.Println(err)
		// 	res.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }
		// res.SetFlashMsg("success", "Logs cleared successfully.")
		// res.Json(w, nil, http.StatusOK)
	}
}
