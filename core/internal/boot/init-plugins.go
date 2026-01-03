//go:build !mono

package boot

import (
	"fmt"
	"log"
	"path/filepath"

	"core/internal/api"
	"core/utils/config"
	"core/utils/env"
	"core/utils/migrate"
	"core/utils/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func InitPlugins(g *api.CoreGlobals) {
	db := g.CoreAPI.SqlDB()
	localPlugins := plugins.LocalPluginSrcDefs()
	systemPlugins := plugins.SystemPluginSrcDefs()

	if env.GO_ENV != env.ENV_PRODUCTION {
		for _, def := range append(systemPlugins, localPlugins...) {
			_, err := plugins.InstallSrcDef(db, def, plugins.InstallOpts{ForceInstall: true, Def: def})
			if err != nil {
				panic(fmt.Sprintf("Error installing plugin %s: %v", def.LocalPath, err))
			}
		}
	}

	// Get current language for translations (only needed if not in dev mode)
	currentLang := "en"
	if env.GO_ENV != env.ENV_DEV {
		cfg, err := config.ReadApplicationConfig()
		if err == nil {
			currentLang = cfg.Lang
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

		// Ensure plugin translations are available for current language (skip in dev mode)
		if env.GO_ENV != env.ENV_DEV {
			if err := sdkutils.EnsureTranslations(dir, currentLang); err != nil {
				log.Printf("Warning: Failed to ensure translations for plugin %s: %v", info.Package, err)
			}
		}

		p := api.NewPluginApi(dir, info, g.GlobalAssets, g.PluginMgr, g.TrafficMgr)
		err = g.PluginMgr.RegisterPlugin(p)
		if err != nil {
			if err := LoadFromBackup(g, info.Package); err != nil {
				g.CoreAPI.Logger().Error(fmt.Sprintf("Error loading from backup: %v", err))
			}

			fmt.Println(dir, " plugin not loaded: ", err)
		}

		migdir := filepath.Join(dir, "resources/migrations")
		if !sdkutils.FsExists(migdir) {
			continue
		}

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

	p := api.NewPluginApi(pkgInstallDir, info, g.GlobalAssets, g.PluginMgr, g.TrafficMgr)
	if err := g.PluginMgr.RegisterPlugin(p); err != nil {
		return err
	}

	plugins.RemoveBackup(pkg)

	return nil
}
