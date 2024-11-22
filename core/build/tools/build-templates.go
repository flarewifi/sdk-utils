package tools

import (
	"core/internal/utils/pkg"
)

func BuildTemplates() {
	pluginDirs := pkg.ListPluginDirs(true)

	for _, p := range pluginDirs {
		pkg.BuildTemplates(p)
	}

}
