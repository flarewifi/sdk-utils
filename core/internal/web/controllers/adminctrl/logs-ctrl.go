package adminctrl

import (
	"core/db/models"
	"core/internal/api"
	logsview "core/resources/views/admin/logs"
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

		searchLogsErr := errors.New(g.CoreAPI.Translate("error", "search_logs_error"))

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
		searchFormTpl, err := g.CoreAPI.HttpAPI.Forms().GetFormTemplate("logs-form", r)
		if err != nil {
			g.CoreAPI.HttpAPI.Response().Error(w, r, searchLogsErr, http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		logsIndex := logsview.Index(g.CoreAPI, result.Logs, searchFormTpl, pagination)

		g.CoreAPI.HttpAPI.Response().AdminView(w, r, sdkapi.ViewPage{
			PageContent: logsIndex,
		})
	}
}

func LogsPostSearch(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		searchForm, err := g.CoreAPI.HttpAPI.Forms().ParseForm("logs-form", w, r)
		if err != nil {
			g.CoreAPI.HttpAPI.Response().Redirect(w, r, "admin:logs:index")
			return
		}

		searchLogsErr := errors.New(g.CoreAPI.Translate("error", "search_logs_error"))

		pkg, err := searchForm.GetStringValue("search", "package")
		if err != nil {
			g.CoreAPI.HttpAPI.Response().Error(w, r, searchLogsErr, http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())

			return
		}

		searchURL := g.CoreAPI.HttpAPI.Helpers().UrlForRoute("admin:logs:index")

		query := url.Values{}

		if pkg != "" {
			query.Add("package", pkg)
		}

		level, err := searchForm.GetStringValue("search", "level")
		if err != nil {
			g.CoreAPI.HttpAPI.Response().Error(w, r, searchLogsErr, http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		if level != "" {
			query.Add("level", level)
		}

		searchTxt, err := searchForm.GetStringValue("search", "search_text")
		if err != nil {
			g.CoreAPI.HttpAPI.Response().Error(w, r, searchLogsErr, http.StatusInternalServerError)
			g.CoreAPI.LoggerAPI.Error(err.Error())
			return
		}

		if searchTxt != "" {
			query.Add("search_text", searchTxt)
		}

		searchURL += "?" + query.Encode()

		http.Redirect(w, r, searchURL, http.StatusSeeOther)
	}
}
