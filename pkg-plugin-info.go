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
	// system_packages are installed but before the plugin files are copied into
	// place; PostInstall runs once the plugin is fully installed. Scripts must
	// guard their own environment (e.g. `command -v opkg || exit 0`) so they
	// no-op safely outside the device.
	PreInstall  string `json:"preinstall"`
	PostInstall string `json:"postinstall"`
}

func GetPluginInfoFromPath(src string) (PluginInfo, error) {
	var info PluginInfo
	if err := JsonRead(filepath.Join(src, "plugin.json"), &info); err != nil {
		return PluginInfo{}, err
	}

	return info, nil
}
