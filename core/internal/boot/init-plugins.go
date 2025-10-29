//go:build !mono

package boot

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"core/internal/api"
	"tools/migrate"
	"tools/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func InitPlugins(g *api.CoreGlobals) {
	db := g.CoreAPI.SqlDB()
	localPlugins := plugins.LocalPluginSrcDefs()
	systemPlugins := plugins.SystemPluginSrcDefs()

	for _, def := range append(systemPlugins, localPlugins...) {
		var info sdkutils.PluginInfo

		installPath, installed := plugins.FindDefInstallPath(def)
		recompile := plugins.NeedsRecompile(def)
		installed = installed && (plugins.ValidateInstallPath(installPath) == nil)
		if installed {
			pluginInfo, err := sdkutils.GetPluginInfoFromPath(installPath)
			if err != nil {
				log.Printf("Error getting plugin info from %q: %s", installPath, err.Error())
				continue
			}
			info = pluginInfo
		}

		toBeRemoved := plugins.IsToBeRemoved(info.Package)
		fmt.Printf("%s is to be removed? %t\n", info.Package, toBeRemoved)

		if toBeRemoved {
			log.Printf("Plugin %q is marked for removal, uninstalling...", info.Package)

			if err := plugins.UninstallPlugin(info.Package, g.CoreAPI.SqlDB()); err != nil {
				log.Printf("Error removing %q plugin: %s", info.Package, err.Error())
			} else {
				log.Printf("Successfully removed %q plugin", info.Package)
				continue
			}
		}

		if plugins.HasPendingUpdate(info.Package) {
			log.Printf("Plugin %q has a pending update, installing...", info.Package)
			err := plugins.MovePendingUpdate(info.Package)
			if err != nil {
				log.Printf("Error installing pending update for %q: %s", info.Package, err.Error())
			} else {
				log.Printf("Successfully installed update for %q", info.Package)
				continue
			}
		}

		// TODO: handle broken plugins
		if installed && !recompile {
			// bp.AppendLog(fmt.Sprintf("Plugin %q is already installed, skipping.", info.Package))
			log.Printf("Plugin %q is already installed, skipping.", info.Package)
			continue
		}

		// create backup, since we are going to reinstall or recompile the plugin
		if installed {
			if err := plugins.CreateBackup(info.Package); err != nil {
				// bp.AppendLog(fmt.Sprintf("Error creating backup for plugin %q: %s", info.Package, err.Error()))
				log.Printf("Error creating backup for plugin %q: %s", info.Package, err.Error())
				continue
			}

			if err := os.RemoveAll(installPath); err != nil {
				// bp.AppendLog(fmt.Sprintf("Error removing plugin %q: %s", info.Package, err.Error()))
				log.Printf("Error removing plugin %q: %s", info.Package, err.Error())
				continue
			}
		}

		info, err := plugins.InstallSrcDef(db, def, plugins.InstallOpts{ForceInstall: true})
		if err != nil {
			// bp.AppendLog(fmt.Sprintf("Error installing plugin %q: %s", def.String(), err.Error()))
			log.Printf("Error installing plugin %q: %s", def.String(), err.Error())
			if plugins.HasBackup(info.Package) {
				// bp.AppendLog(fmt.Sprintf("Restoring backup for plugin %q", info.Package))
				log.Printf("Restoring backup for plugin %q", info.Package)
				if err := plugins.RestoreBackup(info.Package); err != nil {
					// bp.AppendLog(fmt.Sprintf("Error restoring backup for plugin %q: %s", info.Package, err.Error()))
					log.Printf("Error restoring backup for plugin %q: %s", info.Package, err.Error())
				}
			}
		} else {
			// bp.AppendLog(fmt.Sprintf("Successfully installed %q plugin", info.Package))
			log.Printf("Successfully installed %q plugin", info.Package)
			if plugins.HasBackup(info.Package) {
				plugins.RemoveBackup(info.Package)
			}
			if plugins.HasPendingUpdate(info.Package) {
				plugins.RemovePendingUpdate(info.Package)
			}
		}
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

			pkg := filepath.Base(dir)
			if err := LoadFromBackup(g, pkg); err != nil {
				g.CoreAPI.Logger().Error(fmt.Sprintf("Error loading from backup: %v", err))
			}

			continue
		}

		p := api.NewPluginApi(dir, info, g.PluginMgr, g.TrafficMgr)
		err = g.PluginMgr.RegisterPlugin(p)
		if err != nil {
			if err := LoadFromBackup(g, info.Package); err != nil {
				g.CoreAPI.Logger().Error(fmt.Sprintf("Error loading from backup: %v", err))
			}

			fmt.Println(dir, " plugin not loaded: ", err)
		}

		migdir := filepath.Join(dir, "resources/migrations")
		if err := migrate.MigrateUp(db, migdir); err != nil {
			log.Printf("Error in running migration for plugin %s: %+v\n", migdir, err)
		}
	}
}

func LoadFromBackup(g *api.CoreGlobals, pkg string) error {
	if err := plugins.RestoreBackup(pkg); err != nil {
		return fmt.Errorf("%w: Error restoring backup for plugin: %s", err, pkg)
	}

	pkgInstallDir := plugins.GetInstallPath(pkg)
	info, err := sdkutils.GetPluginInfoFromPath(pkgInstallDir)
	if err != nil {
		return fmt.Errorf("error getting plugin info: %w", err)
	}

	p := api.NewPluginApi(pkgInstallDir, info, g.PluginMgr, g.TrafficMgr)
	if err := g.PluginMgr.RegisterPlugin(p); err != nil {
		return err
	}

	plugins.RemoveBackup(pkg)

	return nil
}
