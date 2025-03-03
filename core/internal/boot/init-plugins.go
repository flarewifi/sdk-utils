//go:build !mono

package boot

import (
	"fmt"
	"log"
	"os"

	"core/internal/api"
	"core/internal/utils/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type InstallStatus struct {
	bp *api.BootProgress
}

func (is *InstallStatus) Write(p []byte) (n int, err error) {
	status := string(p)
	is.bp.AppendLog(status)
	return len(p), nil
}

func InitPlugins(g *api.CoreGlobals) {
	bp := g.BootProgress
	db := g.CoreAPI.SqlDb()
	inst := &InstallStatus{bp: bp}

	for _, def := range plugins.AllPluginSrcDefs() {
		var info sdkutils.PluginInfo
		installPath, installed := plugins.FindDefInstallPath(def)
		recompile := plugins.NeedsRecompile(def)
		installed = installed && (plugins.ValidateInstallPath(installPath) == nil)
		if installed {
			pluginInfo, err := sdkutils.GetPluginInfoFromPath(installPath)
			if err != nil {
				bp.AppendLog(fmt.Sprintf("%s: Error getting plugin info: %s", def.String(), err.Error()))
				continue
			}
			info = pluginInfo
		}

		toBeRemoved := plugins.IsToBeRemoved(info.Package)
		fmt.Printf("%s is to be removed? %t\n", info.Package, toBeRemoved)

		if toBeRemoved {
			bp.AppendLog(fmt.Sprintf("%s: Plugin is marked for removal, uninstalling...", info.Package))
			if err := plugins.UninstallPlugin(info.Package, g.CoreAPI.SqlDb()); err != nil {
				bp.AppendLog(fmt.Sprintf("%s: Error removing plugin: %s", info.Package, err.Error()))
			} else {
				bp.AppendLog(fmt.Sprintf("%s: Successfully removed plugin", info.Package))
				continue
			}
		}

		if plugins.HasPendingUpdate(info.Package) {
			bp.AppendLog(fmt.Sprintf("%s: Plugin has a pending update, installing...", info.Package))
			err := plugins.MovePendingUpdate(info.Package)
			if err != nil {
				bp.AppendLog(fmt.Sprintf("%s: Error installing pending update: %s", info.Package, err.Error()))
			} else {
				bp.AppendLog(fmt.Sprintf("%s: Successfully installed update", info.Package))
				continue
			}
		}

		// TODO: handle broken plugins

		if installed && !recompile {
			bp.AppendLog(fmt.Sprintf("%s: Plugin is already installed, skipping.", info.Package))
			continue
		}

		// create backup, since we are going to reinstall or recompile the plugin
		if installed {
			if err := plugins.CreateBackup(info.Package); err != nil {
				bp.AppendLog(fmt.Sprintf("%s: Error creating backup for plugin: %s", info.Package, err.Error()))
				continue
			}

			if err := os.RemoveAll(installPath); err != nil {
				bp.AppendLog(fmt.Sprintf("%s: Error removing plugin: %s", info.Package, err.Error()))
				continue
			}
		}

		info, err := plugins.InstallSrcDef(inst, db, def)
		if err != nil {
			bp.AppendLog(fmt.Sprintf("%s: Error installing plugin: %s", def.String(), err.Error()))
			if plugins.HasBackup(info.Package) {
				bp.AppendLog(fmt.Sprintf("%s: Restoring backup for plugin", info.Package))
				if err := plugins.RestoreBackup(info.Package); err != nil {
					bp.AppendLog(fmt.Sprintf("%s: Error restoring backup for plugin: %s", info.Package, err.Error()))
				}
			}
		} else {
			bp.AppendLog(fmt.Sprintf("%s: Successfully installed plugin", info.Package))
			if plugins.HasBackup(info.Package) {
				plugins.RemoveBackup(info.Package)
			}
			if plugins.HasPendingUpdate(info.Package) {
				plugins.RemovePendingUpdate(info.Package)
			}
		}

		// time.Sleep(1000 * 3 * time.Millisecond)
	}

	// Load plugins
	pluginDirs := plugins.InstalledPluginDirs()
	log.Println("Installed plugin directories:", pluginDirs)
	for _, dir := range pluginDirs {
		log.Println("Loading plugin from :", dir)
		info, err := sdkutils.GetPluginInfoFromPath(dir)
		if err != nil {
			fmt.Println("Error getting plugin info: ", err)
			fmt.Println("Plugin not loaded: ", dir)
			continue
		} else {
			p := api.NewPluginApi(dir, info, g.PluginMgr, g.TrafficMgr)
			g.PluginMgr.RegisterPlugin(p)
		}
	}
}
