package plugins

import (
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func BuildLocalPlugins() error {
	pluginDefs := LocalPluginSrcDefs()
	for _, def := range pluginDefs {
		pluginPath, err := sdkutils.FindPluginSrc(def.LocalPath)
		if err != nil {
			return err
		}

		workdir := filepath.Join(sdkutils.PathTmpDir, "builds", filepath.Base(pluginPath))
		defer os.RemoveAll(workdir)

		if err := PatchPluginDeps(pluginPath); err != nil {
			return err
		}

		if err := BuildTemplates(pluginPath); err != nil {
			return err
		}

		if err := BuildPluginSo(pluginPath, workdir); err != nil {
			return err
		}

		info, err := sdkutils.GetPluginInfoFromPath(pluginPath)
		if err != nil {
			return err
		}

		pluginInstallDir := filepath.Join(sdkutils.PathPluginInstallDir, info.Package)

		if err := os.RemoveAll(pluginInstallDir); err != nil {
			return err
		}

		if err := sdkutils.CopyPluginFiles(pluginPath, pluginInstallDir); err != nil {
			return err
		}

	}
	return nil
}
