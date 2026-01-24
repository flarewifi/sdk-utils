package adminctrl

import (
	"core/internal/api"
	"core/internal/modules/updates"
	powerview "core/resources/views/admin/power"
	cmd "core/utils/shell"
	"fmt"
	"net/http"
	sdkapi "sdk/api"
	"sync/atomic"
	"time"
)

var (
	sysupgradeInProgress atomic.Bool
)

// IsSysupgradeInProgress returns true if a sysupgrade is currently in progress
func IsSysupgradeInProgress() bool {
	return sysupgradeInProgress.Load()
}

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
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		// Check if sysupgrade file is ready
		isSysupgrade := updates.IsSysupgradeReady()

		// Get the no_preserve_data value from form (default to false if not present)
		noPreserve := r.FormValue("no_preserve_data") == "true"

		if isSysupgrade {
			// Set the sysupgrade in progress flag
			sysupgradeInProgress.Store(true)

			// Start the sysupgrade process in background
			go func() {
				time.Sleep(3 * time.Second)
				// Run sysupgrade command with appropriate flags
				// In dev mode, shell.Exec will automatically ignore sysupgrade commands
				cmd.Exec(updates.GetSysupgradeCommand(noPreserve), nil)
				// After completion (or in dev mode immediately), reset the flag
				time.Sleep(2 * time.Second)
				sysupgradeInProgress.Store(false)
			}()

			// Redirect to the sysupgrade progress page
			res.Redirect(w, r, "admin:updates:sysupgrade-progress")
			return
		}

		// Normal reboot (non-sysupgrade)
		go func() {
			time.Sleep(3 * time.Second)
			// In dev mode, shell.Exec will automatically ignore reboot commands
			cmd.Exec("reboot", nil)
		}()

		w.Header().Set("Content-Type", "text/html")
		msg := api.Translate("info", "System is rebooting. Please wait a few minutes before reconnecting")
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
			// In dev mode, shell.Exec will automatically ignore halt commands
			cmd.Exec("halt", nil)
		}()

		w.Header().Set("Content-Type", "text/html")
		msg := g.CoreAPI.Translate("info", "System is shutting down. The device will power off shortly")
		w.Write([]byte(fmt.Sprintf(`<div class="alert alert-success">
	<i class="bi bi-check-circle me-2"></i>
	%s
</div>`, msg)))
	}
}
