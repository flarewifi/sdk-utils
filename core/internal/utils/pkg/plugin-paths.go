package pkg

import (
	"path/filepath"

	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
)

func GetInstallPath(pkg string) string {
	return filepath.Join(sdkpaths.PluginsDir, "installed", pkg)
}

func GetPendingUpdatePath(pkg string) string {
	return filepath.Join(sdkpaths.PluginsDir, "update", pkg)
}

func GetBackupPath(pkg string) string {
	return filepath.Join(sdkpaths.PluginsDir, "backup", pkg)
}

func FindDefInstallPath(def PluginSrcDef) (path string, ok bool) {
	installedPlugins := InstalledPluginsList()
	for _, p := range installedPlugins {
		if (def.Src == PluginSrcLocal || def.Src == PluginSrcSystem) && p.Def.LocalPath == def.LocalPath {
			return p.InstallPath, true
		}
		if def.Src == PluginSrcGit && p.Def.GitURL == def.GitURL {
			return p.InstallPath, true
		}
		if def.Src == PluginSrcStore && p.Def.StorePackage == def.StorePackage {
			return p.InstallPath, true
		}
		if def.Src == PluginSrcZip && p.Def.LocalPath == def.LocalPath {
			return p.InstallPath, true
		}
	}
	return "", false
}

func ListPluginDirs(includeCore bool) []string {
	searchPaths := []string{"plugins/system", "plugins/local"}
	pluginDirs := []string{}

	if includeCore {
		pluginDirs = append(pluginDirs, "core")
	}

	for _, s := range searchPaths {
		var list []string
		if err := sdkfs.LsDirs(s, &list, false); err == nil {
			pluginDirs = append(pluginDirs, list...)
		}
	}

	return pluginDirs
}
