package adminctrl

import (
	"fmt"
	"net/http"
	"strconv"

	"core/db/models"
	"core/internal/api"
	deviceview "core/resources/views/device"
	sdkapi "sdk/api"
)

func DeviceLogsCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		deviceIDStr := r.URL.Query().Get("id")
		if deviceIDStr == "" {
			http.Error(w, "Missing device id parameter", http.StatusBadRequest)
			return
		}

		deviceID, err := strconv.ParseInt(deviceIDStr, 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid device id: %s", deviceIDStr), http.StatusBadRequest)
			return
		}

		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
				page = parsed
			}
		}

		result, err := g.Models.DeviceLog().FindByDeviceIDPaginated(ctx, deviceID, page)
		if err != nil {
			http.Error(w, "Failed to load device logs", http.StatusInternalServerError)
			return
		}

		deviceName := ""
		if dev, err := g.Models.Device().Find(ctx, deviceID); err == nil {
			if dev.Hostname() != "" {
				deviceName = dev.Hostname()
			} else {
				deviceName = dev.MacAddr()
			}
		}

		pagination := g.CoreAPI.UI().Pagination(&sdkapi.UIPaginationOpts{
			PageURL:     g.CoreAPI.Http().Helpers().UrlForRoute("admin:devices:logs"),
			PerPage:     models.DeviceLogsPerPage,
			CurrentPage: page,
			ItemsCount:  result.TotalCount,
			ExtraParams: map[string]string{
				"id": deviceIDStr,
			},
		})

		params := deviceview.DeviceLogsParams{
			Api:        g.CoreAPI,
			DeviceID:   deviceID,
			DeviceName: deviceName,
			Logs:       result.Logs,
			TotalCount: result.TotalCount,
			Pagination: pagination,
		}

		res := g.CoreAPI.HttpAPI.Response()
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: deviceview.DeviceLogsPage(params),
		})
	}
}
