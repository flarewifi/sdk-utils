package sdkutils

import (
	"path/filepath"
)

type PluginInfo struct {
	Name           string   `json:"name"`
	Package        string   `json:"package"`
	Description    string   `json:"description"`
	Version        string   `json:"version"`
	SystemPackages []string `json:"system_packages"`
	SDK            string   `json:"sdk"`

	// PreInstall and PostInstall are optional shell scripts the plugin ships to
	// run install-time routines (e.g. pip-installing libraries not in the opkg
	// feed). Paths are relative to the plugin root. PreInstall runs after the
	// system_packages are installed but before the plugin's Init, so Init can
	// rely on its setup; PostInstall runs after Init (and is skipped if Init
	// fails). Scripts must guard their own environment (e.g.
	// `command -v opkg || exit 0`) so they no-op safely outside the machine.
	PreInstall  string `json:"preinstall"`
	PostInstall string `json:"postinstall"`

	// PreUninstall and PostUninstall are optional shell scripts run when the
	// plugin is removed (core/utils/plugins.UninstallPlugin), for undoing
	// install-time OS-level changes a plugin made OUTSIDE its own install
	// directory (e.g. a system service file, a firewall include, a crontab
	// entry) — installPath's own removal (os.RemoveAll) already cleans up
	// everything inside the plugin's directory, so these are only needed for
	// side effects elsewhere on the filesystem. Paths are relative to the
	// plugin root, same as PreInstall/PostInstall. Both run while the plugin's
	// install directory still exists (that's where the script file itself
	// lives) — PreUninstall first, before the plugin's DB down-migrations and
	// metadata are removed; PostUninstall last, after those but immediately
	// before the install directory itself is deleted. Scripts must guard their
	// own environment and tolerate re-running (e.g. a plugin marked for
	// removal but never actually unloaded before a crash) — there is no
	// version-pinned marker for uninstall scripts the way there is for
	// install scripts, since tracking "did this already run" for a plugin
	// that's gone has no boot to check it against.
	PreUninstall  string `json:"preuninstall"`
	PostUninstall string `json:"postuninstall"`
}

func GetPluginInfoFromPath(src string) (PluginInfo, error) {
	var info PluginInfo
	if err := JsonRead(filepath.Join(src, "plugin.json"), &info); err != nil {
		return PluginInfo{}, err
	}

	return info, nil
}
