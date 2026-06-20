package adminctrl

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"core/internal/api"
	"core/internal/modules/logger"
	logsview "core/resources/views/admin/logs"
	sse "core/utils/sse"
	sdkapi "sdk/api"
)

// sseLogsKey is the dedicated SSE namespace for the admin log tail, so live log
// lines are delivered only to open log-viewer connections (never broadcast to
// portal clients on the shared SSE store).
const sseLogsKey = "admin:logs"

var startLogBroadcasterOnce sync.Once

// logStreamItem is the JSON payload pushed to the live log tail.
type logStreamItem struct {
	DateTime string `json:"datetime"`
	Package  string `json:"package"`
	Level    string `json:"level"`
	Message  string `json:"message"`
	Location string `json:"location"`
}

func LogsIndex(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		pkg := params.Get("package")
		level := params.Get("level")
		searchTxt := params.Get("search_text")

		searchLogsErr := errors.New(g.CoreAPI.Translate("error", "Unable to Search Logs"))

		ipage := atoiOr(params.Get("page"), 1)
		if ipage < 1 {
			ipage = 1
		}
		iPerPage := atoiOr(params.Get("per_page"), 10)
		if iPerPage < 1 {
			iPerPage = 10
		}

		lines, total, err := logger.ReadLogsFiltered(logger.LogFilter{
			Package:    pkg,
			Level:      level,
			SearchText: searchTxt,
			Page:       ipage,
			PerPage:    iPerPage,
		})
		if err != nil {
			g.CoreAPI.HttpAPI.Response().Error(w, r, searchLogsErr, http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		rows := make([]logsview.LogRow, 0, len(lines))
		for _, ll := range lines {
			rows = append(rows, logsview.LogRow{
				DateTime: ll.DateTime,
				Package:  ll.Plugin,
				Level:    levelName(ll.Level),
				Message:  ll.Title,
				Location: ll.Filename + ":" + strconv.Itoa(ll.Line),
			})
		}

		pagination := g.CoreAPI.UI().Pagination(&sdkapi.UIPaginationOpts{
			PageURL:     g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:logs:index"),
			PerPage:     iPerPage,
			CurrentPage: ipage,
			ItemsCount:  int64(total),
			ExtraParams: map[string]string{
				"package":     pkg,
				"level":       level,
				"search_text": searchTxt,
			},
		})

		// Collect package names for the filter: the core plus all installed
		// plugins. (Core logs are tagged with the core package id.)
		packages := []string{"com.flarego.core"}
		for _, p := range g.PluginMgr.Plugins() {
			packages = append(packages, p.Info().Package)
		}

		searchData := logsview.LogsSearchData{
			Packages:   packages,
			Package:    pkg,
			Level:      level,
			SearchText: searchTxt,
			ActionURL:  g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:logs:search"),
			StreamURL:  g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:logs:stream"),
		}

		logsIndex := logsview.Index(g.CoreAPI, rows, searchData, pagination)

		g.CoreAPI.HttpAPI.Response().AdminView(w, r, sdkapi.ViewPage{
			PageContent: logsIndex,
		})
	}
}

func LogsPostSearch(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			g.CoreAPI.HttpAPI.Response().Redirect(w, r, "admin:logs:index")
			return
		}

		// Clear the rotating log file.
		if r.FormValue("clear_logs") == "1" {
			if err := logger.ClearLogs(); err != nil {
				g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(g.CoreAPI.Translate("error", "Unable to clear logs")), http.StatusInternalServerError)
				g.CoreAPI.LoggerAPI.Error(err.Error())
				return
			}
			http.Redirect(w, r, g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:logs:index"), http.StatusSeeOther)
			return
		}

		// Otherwise apply filters via query params on the index page.
		query := url.Values{}
		if pkg := r.FormValue("package"); pkg != "" {
			query.Add("package", pkg)
		}
		if level := r.FormValue("level"); level != "" {
			query.Add("level", level)
		}
		if searchTxt := r.FormValue("search_text"); searchTxt != "" {
			query.Add("search_text", searchTxt)
		}

		searchURL := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:logs:index")
		if encoded := query.Encode(); encoded != "" {
			searchURL += "?" + encoded
		}
		http.Redirect(w, r, searchURL, http.StatusSeeOther)
	}
}

// LogsStream is the SSE endpoint for the live log tail. Each connection receives
// every subsequently emitted log line as a "log" event.
func LogsStream(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := g.CoreAPI.HttpAPI.Auth().CurrentAcct(r); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		startLogBroadcasterOnce.Do(startLogBroadcaster)

		s, err := sse.NewSocket(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sse.AddSocket(sseLogsKey, s)
		s.Listen()
	}
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// startLogBroadcaster subscribes to the logger once for the process lifetime and
// fans out each new line to all open log-viewer SSE connections.
func startLogBroadcaster() {
	ch, _ := logger.Subscribe()
	go func() {
		for ll := range ch {
			data, err := json.Marshal(logStreamItem{
				DateTime: ll.DateTime,
				Package:  ll.Plugin,
				Level:    levelName(ll.Level),
				Message:  ll.Title,
				Location: ll.Filename + ":" + strconv.Itoa(ll.Line),
			})
			if err != nil {
				continue
			}
			sse.Emit(sseLogsKey, "log", data)
		}
	}()
}

func levelName(level int) string {
	switch level {
	case 1:
		return "debug"
	case 2:
		return "error"
	default:
		return "info"
	}
}

func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
