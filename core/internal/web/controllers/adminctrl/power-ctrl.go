package adminctrl

import (
	"core/internal/api"
	"core/internal/modules/updates"
	powerview "core/resources/views/admin/power"
	cmd "core/utils/shell"
	"errors"
	"fmt"
	"net/http"
	sdkapi "sdk/api"
	"sync/atomic"
	"time"
)

var (
	sysupgradeInProgress atomic.Bool
	sysupgradeError      atomic.Pointer[error]
)

// IsSysupgradeInProgress returns true if a sysupgrade is currently in progress
func IsSysupgradeInProgress() bool {
	return sysupgradeInProgress.Load()
}

// SysupgradeError returns the error from the last sysupgrade attempt, if it failed.
// It is nil while a sysupgrade is in progress and after a successful one — a
// successful sysupgrade reboots the device before this code ever resumes, so a
// non-nil value here always means the flash failed and the device is still on its
// current firmware.
func SysupgradeError() error {
	v := sysupgradeError.Load()
	if v == nil {
		return nil
	}
	return *v
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
			// Set the sysupgrade in progress flag and clear any previous failure
			sysupgradeInProgress.Store(true)
			sysupgradeError.Store(nil)

			// Start the sysupgrade process in background
			go func() {
				time.Sleep(3 * time.Second)

				defer func() {
					// After completion (or in dev mode immediately), reset the flag
					time.Sleep(2 * time.Second)
					sysupgradeInProgress.Store(false)
				}()

				// Run sysupgrade command with appropriate flags
				// In dev mode, shell.Exec will automatically ignore sysupgrade commands
				if err := cmd.Exec(updates.GetSysupgradeCommand(noPreserve), nil); err != nil {
					api.LoggerAPI.Error(fmt.Sprintf("failed to execute sysupgrade command: %v", err))
					// Store a generic, translated message for the progress page's poll to
					// display — never the raw exec error (it can contain command/file paths).
					failMsg := errors.New(api.Translate("error", "Firmware upgrade failed. The device was not flashed and remains on its current firmware"))
					sysupgradeError.Store(&failMsg)
				}
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
		w.Write([]byte(fmt.Sprintf(`<div id="shutdown-alert" class="alert alert-success">
	<i class="bi bi-check-circle me-2"></i>
	%s
</div>`, msg)))
	}
}
