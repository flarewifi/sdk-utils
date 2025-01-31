package tools

import (
	"core/internal/utils/pkg"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func BuildQueries() {
	pluginDirs := []string{}

	defs := pkg.AllPluginSrcDefs()
	for _, def := range defs {
		if def.Src == sdkutils.PluginSrcLocal || def.Src == sdkutils.PluginSrcSystem {
			pluginDirs = append(pluginDirs, def.LocalPath)
		}
	}

	corePath := filepath.Join(sdkutils.PathAppDir, "core")
	pluginDirs = append(pluginDirs, corePath)

	for _, p := range pluginDirs {
		pkg.BuildQueries(p)
	}
}
