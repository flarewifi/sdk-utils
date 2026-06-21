package boot

import (
	"fmt"

	"core/internal/api"
	"core/utils/plugins"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// Plugin lifecycle scripts (preinstall/postinstall) are normally run by the
// install path (plugins.InstallPlugin / InstallPrebuilt), which also records a
// version-pinned marker. os_image builds, however, bake plugins straight into
// plugins/installed and never go through that path — so on a device's first boot
// the markers are absent and the scripts must run later.
//
// Both scripts (and a plugin's system_packages) need the network, so they are no
// longer run inline during boot. The online monitor's internet-up provisioning
// pass (ProvisionInstalledPlugins in provision.go) drives them once connectivity
// is available, using the shared runInstallScriptOnce helper below. Every
// subsequent boot finds a marker matching the installed version and skips them; a
// plugin update bumps the version, invalidating the marker so the new version's
// scripts run once on the next internet-up.

// runInstallScriptOnce runs a single plugin install script (preinstall or
// postinstall) exactly once per plugin version, recording a version-pinned marker
// on success. Failures are logged, never fatal: a misbehaving script must not
// keep the device from booting or break the provisioning pass for other plugins.
func runInstallScriptOnce(g *api.CoreGlobals, dir string, info sdkutils.PluginInfo, scriptRel, phase string) {
	if scriptRel == "" {
		return
	}

	// Already ran for this exact version — nothing to do. A version mismatch
	// (fresh install or upgrade) falls through and re-runs.
	if plugins.ReadScriptMarker(info.Package, phase) == info.Version {
		return
	}

	// RunInstallScript records the version-pinned marker on success, so a
	// completed script is not retried on the next boot.
	if err := plugins.RunInstallScript(dir, info, scriptRel, phase); err != nil {
		if logErr := g.CoreAPI.Logger().Error(fmt.Sprintf("plugin %q %s script failed: %v", info.Package, phase, err)); logErr != nil {
		}
	}
}
