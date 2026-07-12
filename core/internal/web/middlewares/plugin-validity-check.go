package middlewares

import (
	"errors"
	"net/http"

	"core/utils/plugins"
	sdkapi "sdk/api"
)

// PluginValidityCheck blocks a plugin's HTTP routes once it becomes invalid
// (blocked, disabled, update-skipped, or queued for uninstall -- see
// plugins.IsInvalid) WITHOUT waiting for the next reboot. A Go plugin .so
// can't be unloaded mid-process, so a plugin blocked/disabled at runtime
// (e.g. the daily cloud denylist sync, or a lapsed store purchase caught by
// boot.ValidateStorePlugins' hourly recheck) previously kept serving its
// pages until the machine restarted. Checked per-request against
// plugins.LoadValidityCache's in-memory registry rather than a live
// filesystem stat -- a loaded plugin .so shares the host process's own
// filesystem privileges, so re-reading the marker file on every request would
// let a compromised plugin simply delete its own marker to bypass this check.
func PluginValidityCheck(api sdkapi.IPluginApi, pkg string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if plugins.IsInvalid(pkg) {
				err := errors.New(api.Translate("error", "This plugin is no longer available"))
				api.Http().Response().Error(w, r, err, http.StatusServiceUnavailable)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
