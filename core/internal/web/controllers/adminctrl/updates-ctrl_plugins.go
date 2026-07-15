//go:build !mono

// Plugin listing for the core Software Updates page (non-mono only).
//
// Plugins are not updated independently. The unified "Check for updates" button
// renders the list of installed plugins (renderPluginUpdatesOOB) so the user can
// see which ones have newer versions. "Upgrade Now" then applies everything that
// is outdated together: a core update rebuilds every plugin against the new core
// (model A, system-update.go), while a plugin-only update (core current) re-stages
// the latest plugins against the unchanged core. There are no per-plugin endpoints.
package adminctrl

import (
	"core/internal/api"
	"core/internal/modules/updates"
	updatesview "core/resources/views/admin/updates"
	"fmt"
	"net/http"
)

// checkPluginUpdatesList returns every installed plugin with its update status
// from the cloud (CurrentVersion + LatestVersion/HasUpdate filled). Best-effort:
// on error it returns nil so the check still shows the core result. Used by the
// unified "Check for updates" to decide whether to offer "Upgrade Now" and to
// refresh the plugin list, from a single cloud lookup.
func checkPluginUpdatesList(g *api.CoreGlobals) []updates.PluginUpdate {
	list, err := updates.CheckPluginUpdates(g)
	if err != nil {
		g.CoreAPI.LoggerAPI.Error(fmt.Sprintf("check plugin updates: %v", err))
		return nil
	}
	return list
}

// renderPluginUpdatesOOB renders the (already fetched) plugin list as an
// out-of-band swap so the unified "Check for updates" button refreshes both the
// core result and the plugin list in a single response. coreCurrentVersion/
// coreNewVersion surface the pending core update (if any) as the first row of the
// same list, alongside the plugin versions; pass coreNewVersion == "" when there is
// no core update.
func renderPluginUpdatesOOB(g *api.CoreGlobals, w http.ResponseWriter, r *http.Request, list []updates.PluginUpdate, coreCurrentVersion string, coreNewVersion string) {
	if err := updatesview.PluginUpdatesListPartial(g.CoreAPI, coreCurrentVersion, coreNewVersion, list).Render(r.Context(), w); err != nil {
		g.CoreAPI.LoggerAPI.Error(err.Error())
	}
}

// hasPluginUpdates reports whether any installed plugin has a newer version
// available. Used by the download page to decide whether a plugin-only upgrade
// should proceed when there is no core update. Best-effort: nil/empty → false.
func hasPluginUpdates(g *api.CoreGlobals) bool {
	return len(outdatedPlugins(checkPluginUpdatesList(g))) > 0
}

// startPluginOnlyDownload stages the latest plugins against the current core (no
// core change) in the background.
func startPluginOnlyDownload(g *api.CoreGlobals) {
	updates.StagePluginsUpdate(g)
}
