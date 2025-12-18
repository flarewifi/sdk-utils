package adminctrl

import (
	"core/internal/api"
	"core/internal/utils/updates"
	powerview "core/resources/views/admin/power"
	"fmt"
	"net/http"
	sdkapi "sdk/api"
	"time"
	cmd "core/tools/shell"
)

func RebootPageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		page := powerview.RebootPage(g.CoreAPI)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})
	}
}

func RebootCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if sysupgrade file is ready
		isSysupgrade := updates.IsSysupgradeReady()

		go func() {
			time.Sleep(3 * time.Second)
			if isSysupgrade {
				// Run sysupgrade command (this will flash firmware and reboot)
				cmd.Exec("sysupgrade "+updates.GetSysupgradePath(), nil)
			} else {
				// Normal reboot
				cmd.Exec("reboot", nil)
			}
		}()

		w.Header().Set("Content-Type", "text/html")
		var msg string
		if isSysupgrade {
			msg = g.CoreAPI.Translate("info", "Firmware upgrade in progress. Do not power off the device")
		} else {
			msg = g.CoreAPI.Translate("info", "System is rebooting Please wait a few minutes before reconnecting")
		}
		w.Write([]byte(fmt.Sprintf(`<div id="notification-area" class="alert alert-info mb-4">
		<i class="bi bi-info-circle me-2"></i>
		%s
	</div>`, msg)))
	}
}

func ShutdownPageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := g.CoreAPI.HttpAPI.Response()
		page := powerview.ShutdownPage(g.CoreAPI)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})
	}
}

func ShutdownCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		go func() {
			time.Sleep(3 * time.Second)
			cmd.Exec("halt", nil)
		}()

		w.Header().Set("Content-Type", "text/html")
		msg := g.CoreAPI.Translate("info", "System is shutting down The device will power off shortly")
		w.Write([]byte(fmt.Sprintf(`<div class="alert alert-success">
			<i class="bi bi-check-circle me-2"></i>
			%s
		</div>`, msg)))
	}
}
