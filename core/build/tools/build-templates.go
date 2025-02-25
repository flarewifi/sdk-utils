package tools

import (
	"core/internal/utils/plugins"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func BuildTemplates() {
	pluginDirs := []string{}

	defs := plugins.AllPluginSrcDefs()
	for _, def := range defs {
		if def.Src == sdkutils.PluginSrcLocal || def.Src == sdkutils.PluginSrcSystem {
			pluginDirs = append(pluginDirs, def.LocalPath)
		}
	}

	corePath := filepath.Join(sdkutils.PathAppDir, "core")
	pluginDirs = append(pluginDirs, corePath)

	for _, p := range pluginDirs {
		plugins.BuildTemplates(p)
	}
}
