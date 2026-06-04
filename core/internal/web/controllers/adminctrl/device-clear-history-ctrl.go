package adminctrl

import (
	"fmt"
	"net/http"
	"strconv"

	"core/internal/api"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func DeviceClearHistoryCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		res := g.CoreAPI.HttpAPI.Response()

		deviceIDStr := r.FormValue("id")
		deviceID, err := strconv.ParseInt(deviceIDStr, 10, 64)
		if err != nil {
			res.FlashMsg(w, r, "Invalid device ID", sdkapi.FlashMsgError)
			http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
			return
		}

		redirectURL := g.CoreAPI.Http().Helpers().UrlForRoute("admin:devices:diag") + fmt.Sprintf("?id=%d", deviceID)

		// Delete fingerprints
		_, err = g.Database.DB.ExecContext(ctx, "DELETE FROM device_fingerprints WHERE device_id = ?", deviceID)
		if err != nil {
			res.FlashMsg(w, r, "Failed to clear device history", sdkapi.FlashMsgError)
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}

		// Delete non-current MACs
		_, err = g.Database.DB.ExecContext(ctx, "DELETE FROM device_macs WHERE device_id = ? AND is_current = 0", deviceID)
		if err != nil {
			res.FlashMsg(w, r, "Failed to clear device history", sdkapi.FlashMsgError)
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}

		// Regenerate cookie_token
		newToken := sdkutils.NewUUID()
		_, err = g.Database.DB.ExecContext(ctx, "UPDATE devices SET cookie_token = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", newToken, deviceID)
		if err != nil {
			res.FlashMsg(w, r, "Failed to clear device history", sdkapi.FlashMsgError)
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}

		// Log the action
		metadata := map[string]interface{}{
			"cleared": "fingerprints, non-current MACs, cookie_token regenerated",
		}
		_ = g.Models.DeviceLog().Create(ctx, deviceID, "Device history cleared by admin", metadata)

		res.FlashMsg(w, r, "Device history cleared", sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}
