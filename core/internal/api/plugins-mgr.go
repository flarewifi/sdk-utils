package api

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	sdkplugin "sdk/api"

	"core/db"
	"core/db/models"
	"core/internal/network"
	"core/internal/sessmgr"
	"core/tools/config"
	"core/tools/migrate"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func NewPluginMgr(d *db.Database, m *models.Models, paymgr *PaymentsMgr, clntReg *sessmgr.ClientRegister, clntMgr *sessmgr.SessionsMgr, trfkMgr *network.TrafficMgr) *PluginsMgr {
	pmgr := &PluginsMgr{
		db:      d,
		models:  m,
		paymgr:  paymgr,
		clntReg: clntReg,
		clntMgr: clntMgr,
		plugins: []*PluginApi{},
	}
	return pmgr
}

type PluginsMgr struct {
	CoreAPI *PluginApi
	db      *db.Database
	models  *models.Models
	paymgr  *PaymentsMgr
	clntReg *sessmgr.ClientRegister
	clntMgr *sessmgr.SessionsMgr
	trfkMgr *network.TrafficMgr
	plugins []*PluginApi
}

func (self *PluginsMgr) InitCoreApi(coreApi *PluginApi) {
	self.CoreAPI = coreApi
	coreApi.Initialize(coreApi)
	coreApi.LoadAssetsManifest()
	self.plugins = append(self.plugins, coreApi)
}

func (self *PluginsMgr) Plugins() []*PluginApi {
	return self.plugins
}

func (self *PluginsMgr) FindByName(name string) (sdkplugin.IPluginApi, bool) {
	for _, p := range self.plugins {
		if p.Info().Name == name {
			return p, true
		}
	}
	return nil, false
}

func (self *PluginsMgr) FindByPkg(pkg string) (sdkplugin.IPluginApi, bool) {
	for _, p := range self.plugins {
		if p.Info().Package == pkg {
			return p, true
		}
	}
	return nil, false
}

func (self *PluginsMgr) All() []sdkplugin.IPluginApi {
	plugins := []sdkplugin.IPluginApi{}
	for _, p := range self.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

func (self *PluginsMgr) PaymentMethods() []sdkplugin.IPluginApi {
	methods := []sdkplugin.IPluginApi{}
	for _, p := range self.plugins {
		pmnt := p.Payments().(*PaymentsApi)
		if pmnt.paymentsMgr != nil {
			methods = append(methods, p)
		}
	}
	return methods
}

func (self *PluginsMgr) GetAdminTheme() (*PluginApi, *ThemesApi, error) {
	cfg, err := config.ReadThemesConfig()
	if err != nil {
		return nil, nil, err
	}

	pkg := cfg.AdminThemePkg
	p, ok := self.FindByPkg(pkg)
	if !ok {
		return nil, nil, fmt.Errorf("admin theme plugin '%s' is not installed", pkg)
	}

	themeApi := p.Themes().(*ThemesApi)
	if themeApi.AdminTheme == nil {
		return nil, nil, fmt.Errorf("plugin '%s' doesn't implement theme API", pkg)
	}

	return p.(*PluginApi), themeApi, nil
}

func (self *PluginsMgr) GetPortalTheme() (*PluginApi, *ThemesApi, error) {
	cfg, err := config.ReadThemesConfig()
	if err != nil {
		return nil, nil, err
	}

	pkg := cfg.PortalThemePkg
	p, ok := self.FindByPkg(pkg)
	if !ok {
		return nil, nil, fmt.Errorf("portal theme plugin '%s' is not installed", pkg)
	}

	themeApi := p.Themes().(*ThemesApi)
	if themeApi.PortalTheme == nil {
		return nil, nil, fmt.Errorf("plugin '%s' doesn't implement theme API", pkg)
	}

	return p.(*PluginApi), themeApi, nil
}

// RerunPluginMigrations re-runs migrations for all loaded plugins
// This is used after database reset to recreate plugin tables
func (self *PluginsMgr) RerunPluginMigrations(newDB *sql.DB) error {
	log.Println("Re-running plugin migrations after database reset...")

	for _, p := range self.plugins {
		// Skip core plugin (already migrated)
		if p.Info().Package == "com.flarego.core" {
			continue
		}

		pluginDir := p.Dir()
		migdir := filepath.Join(pluginDir, "resources/migrations")

		if !sdkutils.FsExists(migdir) {
			log.Printf("No migrations directory for plugin %s, skipping...", p.Info().Package)
			continue
		}

		log.Printf("Running migrations for plugin: %s", p.Info().Package)
		// Pass the plugin directory, not the migrations subdirectory
		// This ensures proper temp directory isolation per plugin
		if err := migrate.MigrateUp(newDB, pluginDir); err != nil {
			return fmt.Errorf("failed to run migrations for plugin %s: %w", p.Info().Package, err)
		}
		log.Printf("Successfully ran migrations for plugin: %s", p.Info().Package)
	}

	log.Println("All plugin migrations completed successfully")
	return nil
}
