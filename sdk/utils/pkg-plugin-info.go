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
}

func GetPluginInfoFromPath(src string) (PluginInfo, error) {
	var info PluginInfo
	if err := JsonRead(filepath.Join(src, "plugin.json"), &info); err != nil {
		return PluginInfo{}, err
	}

	return info, nil
}
