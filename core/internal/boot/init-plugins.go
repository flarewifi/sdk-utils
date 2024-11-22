//go:build !mono

package boot

import (
	"fmt"
	"log"
	"os"
	sdkplugin "sdk/api/plugin"

	"core/internal/plugins"
	"core/internal/utils/pkg"
)

type InstallStatus struct {
	bp *plugins.BootProgress
}

func (is *InstallStatus) Write(p []byte) (n int, err error) {
	status := string(p)
	is.bp.AppendLog(status)
	return len(p), nil
}

func InitPlugins(g *plugins.CoreGlobals) {
	bp := g.BootProgress
	inst := &InstallStatus{bp: bp}

	for _, def := range pkg.AllPluginDef() {
		var info sdkplugin.PluginInfo
		path, installed := pkg.FindDefInstallPath(def)
		recompile := pkg.NeedsRecompile(def)
		installed = installed && (pkg.ValidateInstallPath(path) == nil)
		if installed {
			info, _ = pkg.GetSrcInfo(path)
		}

		if pkg.IsToBeRemoved(info.Package) {
			bp.AppendLog(fmt.Sprintf("%s: Plugin is marked for removal, uninstalling...", info.Package))
			if err := pkg.RemovePlugin(info.Package); err != nil {
				bp.AppendLog(fmt.Sprintf("%s: Error removing plugin: %s", info.Package, err.Error()))
			} else {
				bp.AppendLog(fmt.Sprintf("%s: Successfully removed plugin", info.Package))
				continue
			}
		}

		if pkg.HasPendingUpdate(info.Package) {
			bp.AppendLog(fmt.Sprintf("%s: Plugin has a pending update, installing...", info.Package))
			err := pkg.MovePendingUpdate(info.Package)
			if err != nil {
				bp.AppendLog(fmt.Sprintf("%s: Error installing pending update: %s", info.Package, err.Error()))
			} else {
				bp.AppendLog(fmt.Sprintf("%s: Successfully installed update", info.Package))
				continue
			}
		}

		// TODO: handle broken plugins

		if installed && !recompile {
			bp.AppendLog(fmt.Sprintf("%s: Plugin is already installed", info.Package))
			continue
		}

		if installed {
			if err := pkg.CreateBackup(info.Package); err != nil {
				bp.AppendLog(fmt.Sprintf("%s: Error creating backup for plugin: %s", info.Package, err.Error()))
				continue
			}

			if err := os.RemoveAll(path); err != nil {
				bp.AppendLog(fmt.Sprintf("%s: Error removing plugin: %s", info.Package, err.Error()))
				continue
			}
		}

		info, err := pkg.InstallSrcDef(inst, def)
		if err != nil {
			bp.AppendLog(fmt.Sprintf("%s: Error installing plugin: %s", def.String(), err.Error()))
			if pkg.HasBackup(info.Package) {
				bp.AppendLog(fmt.Sprintf("%s: Restoring backup for plugin", info.Package))
				if err := pkg.RestoreBackup(info.Package); err != nil {
					bp.AppendLog(fmt.Sprintf("%s: Error restoring backup for plugin: %s", info.Package, err.Error()))
				}
			}
		} else {
			bp.AppendLog(fmt.Sprintf("%s: Successfully installed plugin", info.Package))
			if pkg.HasBackup(info.Package) {
				pkg.RemoveBackup(info.Package)
			}
			if pkg.HasPendingUpdate(info.Package) {
				pkg.RemovePendingUpdate(info.Package)
			}
		}

		// time.Sleep(1000 * 3 * time.Millisecond)
	}

	// Load plugins
	pluginDirs := pkg.InstalledDirList()
	log.Println("Installed plugin directories:", pluginDirs)
	for _, dir := range pluginDirs {
		log.Println("Loading plugin from :", dir)
		p := plugins.NewPluginApi(dir, g.PluginMgr, g.TrafficMgr)
		g.PluginMgr.RegisterPlugin(p)
	}
}
