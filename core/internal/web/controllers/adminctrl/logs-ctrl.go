package adminctrl

import (
	"core/db/models"
	"core/internal/api"
	logsview "core/resources/views/admin/logs"
	"core/utils/config"
	"errors"
	"net/http"
	"net/url"
	sdkapi "sdk/api"
	"strconv"
)

func LogsIndex(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		pkg := params.Get("package")
		level := params.Get("level")
		page := params.Get("page")
		perPage := params.Get("per_page")
		searchTxt := params.Get("search_text")

		var ipage, iPerPage int
		var err error

		searchLogsErr := errors.New(g.CoreAPI.Translate("error", "Unable to Search Logs"))

		if page != "" {
			ipage, err = strconv.Atoi(page)
			if err != nil {
				g.CoreAPI.HttpAPI.Response().Error(w, r, searchLogsErr, http.StatusInternalServerError)
				g.CoreAPI.LoggerAPI.Error(err.Error())
				return
			}
		}
		if ipage == 0 {
			ipage = 1
		}

		if perPage != "" {
			iPerPage, err = strconv.Atoi(perPage)
			if err != nil {
				g.CoreAPI.HttpAPI.Response().Error(w, r, searchLogsErr, http.StatusInternalServerError)
				g.CoreAPI.LoggerAPI.Error(err.Error())
				return
			}
		}
		if iPerPage == 0 {
			iPerPage = 10
		}

		opts := models.LogsPaginateOpts{
			Page:       ipage,
			PerPage:    iPerPage,
			Package:    pkg,
			Level:      level,
			SearchText: searchTxt,
		}

		result, err := g.Models.Log().Paginate(r.Context(), opts)
		if err != nil {
			g.CoreAPI.HttpAPI.Response().Error(w, r, searchLogsErr, http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		pagination := g.CoreAPI.UI().Pagination(&sdkapi.UIPaginationOpts{
			PageURL:     g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:logs:index"),
			PerPage:     iPerPage,
			CurrentPage: ipage,
			ItemsCount:  result.Count,
			ExtraParams: map[string]string{
				"package":     pkg,
				"level":       level,
				"search_text": searchTxt,
			},
		})

		// Collect package names from all plugins
		var packages []string
		pkgs := g.PluginMgr.All()
		for _, p := range pkgs {
			info := p.Info()
			packages = append(packages, info.Package)
		}

		// Get log settings from application config
		logsRetentionDays := 3 // default
		enableLogging := false // default disabled
		appCfg, err := config.ReadApplicationConfig()
		if err == nil {
			if appCfg.LogsRetentionDays > 0 {
				logsRetentionDays = appCfg.LogsRetentionDays
			}
			enableLogging = appCfg.EnableLogging
		}

		searchData := logsview.LogsSearchData{
			Packages:          packages,
			Package:           pkg,
			Level:             level,
			SearchText:        searchTxt,
			ActionURL:         g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:logs:search"),
			LogsRetentionDays: logsRetentionDays,
			EnableLogging:     enableLogging,
		}

		logsIndex := logsview.Index(g.CoreAPI, result.Logs, searchData, pagination)

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

		pkg := r.FormValue("package")
		level := r.FormValue("level")
		searchTxt := r.FormValue("search_text")

		searchURL := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:logs:index")

		query := url.Values{}

		if pkg != "" {
			query.Add("package", pkg)
		}

		if level != "" {
			query.Add("level", level)
		}

		if searchTxt != "" {
			query.Add("search_text", searchTxt)
		}

		searchURL += "?" + query.Encode()

		// Handle save_settings action (includes enable_logging and retention days)
		if r.FormValue("save_settings") == "1" {
			retentionDaysStr := r.FormValue("logs_retention_days")
			retentionDays, err := strconv.Atoi(retentionDaysStr)
			if err != nil || (retentionDays != 3 && retentionDays != 7 && retentionDays != 14 && retentionDays != 30) {
				g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(g.CoreAPI.Translate("error", "Invalid retention period")), http.StatusBadRequest)
				return
			}

			// Read current config
			appCfg, err := config.ReadApplicationConfig()
			if err != nil {
				g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(g.CoreAPI.Translate("error", "Unable to read configuration")), http.StatusInternalServerError)
				g.CoreAPI.LoggerAPI.Error(err.Error())
				return
			}

			// Update settings
			appCfg.LogsRetentionDays = retentionDays
			appCfg.EnableLogging = r.FormValue("enable_logging") == "1"

			// Save config
			if err := config.WriteApplicationConfig(appCfg); err != nil {
				g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(g.CoreAPI.Translate("error", "Unable to save configuration")), http.StatusInternalServerError)
				g.CoreAPI.LoggerAPI.Error(err.Error())
				return
			}

			successMsg := g.CoreAPI.Translate("success", "Log settings saved successfully")
			g.CoreAPI.HttpAPI.Response().FlashMsg(w, r, successMsg, sdkapi.FlashMsgSuccess)
			http.Redirect(w, r, g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:logs:index"), http.StatusSeeOther)
			return
		}

		// Handle clear_logs action
		if r.FormValue("clear_logs") == "1" {
			if err := g.Models.Log().Clear(r.Context()); err != nil {
				g.CoreAPI.HttpAPI.Response().Error(w, r, errors.New(g.CoreAPI.Translate("error", "Unable to clear logs")), http.StatusInternalServerError)
				g.CoreAPI.LoggerAPI.Error(err.Error())
				return
			}
			http.Redirect(w, r, g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:logs:index"), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, searchURL, http.StatusSeeOther)
	}
}
