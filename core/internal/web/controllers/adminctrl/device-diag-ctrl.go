package adminctrl

import (
	"fmt"
	"net/http"
	"strconv"

	"core/db/queries"
	"core/internal/api"
	deviceview "core/resources/views/device"
	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func DeviceDiagCtrl(g *api.CoreGlobals) http.HandlerFunc {
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

		dev, err := g.Models.Device().Find(ctx, deviceID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Device not found: %d", deviceID), http.StatusNotFound)
			return
		}

		macs, err := g.Models.DeviceMac().FindByDeviceID(ctx, dev.ID())
		if err != nil {
			macs = nil
		}

		fingerprints, err := g.Models.DeviceFingerprint().FindByDeviceID(ctx, dev.ID())
		if err != nil {
			fingerprints = nil
		}

		logsResult, err := g.Models.DeviceLog().FindByDeviceIDPaginated(ctx, dev.ID(), 1)
		var recentLogs []queries.DeviceLog
		var totalLogs int64
		if err == nil {
			recentLogs = logsResult.Logs
			totalLogs = logsResult.TotalCount
		}

		params := deviceview.DeviceDiagParams{
			Api:          g.CoreAPI,
			DeviceID:     dev.ID(),
			UUID:         dev.UUID(),
			CookieToken:  dev.CookieToken(),
			Ipv4Addr:     dev.Ipv4Addr(),
			Ipv6Addr:     dev.Ipv6Addr(),
			MacAddr:      dev.MacAddr(),
			Hostname:     dev.Hostname(),
			Status:       int(dev.Status()),
			CreatedAt:    sdkutils.UtcToLocalTime(dev.CreatedAt()).Format("2006-01-02 15:04:05"),
			UpdatedAt:    sdkutils.UtcToLocalTime(dev.UpdatedAt()).Format("2006-01-02 15:04:05"),
			Macs:         macs,
			Fingerprints: fingerprints,
			RecentLogs:   recentLogs,
			TotalLogs:    totalLogs,
		}

		res := g.CoreAPI.HttpAPI.Response()
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: deviceview.DeviceDiagPage(params),
		})
	}
}
