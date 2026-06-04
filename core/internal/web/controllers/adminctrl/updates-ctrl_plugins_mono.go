//go:build mono

// In a mono build the core and plugins are compiled together and updated as one
// system release, so there is no per-plugin update list or plugin-only upgrade.
// These twins exist only so the shared controllers in updates-ctrl.go can call
// them unconditionally; here they are no-ops.
package adminctrl

import (
	"core/internal/api"
	"core/internal/modules/updates"
	"net/http"
)

func checkPluginUpdatesList(g *api.CoreGlobals) []updates.PluginUpdate { return nil }

func renderPluginUpdatesOOB(g *api.CoreGlobals, w http.ResponseWriter, r *http.Request, list []updates.PluginUpdate) {
}

// hasPluginUpdates is always false on mono — there is no plugin-only upgrade path.
func hasPluginUpdates(g *api.CoreGlobals) bool { return false }

// startPluginOnlyDownload is a no-op on mono — plugins ship with the core.
func startPluginOnlyDownload(g *api.CoreGlobals) {}
