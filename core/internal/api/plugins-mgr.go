package api

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	sdkplugin "sdk/api"

	"core/db"
	"core/db/models"
	"core/internal/events"
	"core/internal/modules/ubus"
	"core/internal/network"
	"core/internal/sessmgr"
	"core/utils/config"
	"core/utils/migrate"
	"core/utils/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func NewPluginMgr(d *db.Database, m *models.Models, paymgr *PaymentsMgr, clntReg *sessmgr.ClientRegister, clntMgr *sessmgr.SessionsMgr, trfkMgr *network.TrafficMgr, eventsMgr *events.EventsManager) *PluginsMgr {
	pmgr := &PluginsMgr{
		db:        d,
		models:    m,
		paymgr:    paymgr,
		clntReg:   clntReg,
		clntMgr:   clntMgr,
		trfkMgr:   trfkMgr,
		eventsMgr: eventsMgr,
		plugins:   []*PluginApi{},
	}
	return pmgr
}

type PluginsMgr struct {
	CoreAPI      *PluginApi
	db           *db.Database
	models       *models.Models
	paymgr       *PaymentsMgr
	clntReg      *sessmgr.ClientRegister
	clntMgr      *sessmgr.SessionsMgr
	trfkMgr      *network.TrafficMgr
	eventsMgr    *events.EventsManager
	plugins      []*PluginApi
	globalAssets *GlobalAssets
	wifiMgr      *ubus.WifiMgr
}

// SetDeps stores the dependencies needed for live plugin registration after a store install.
func (self *PluginsMgr) SetDeps(assets *GlobalAssets, wifiMgr *ubus.WifiMgr) {
	self.globalAssets = assets
	self.wifiMgr = wifiMgr
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

// GetAdminTheme returns the active admin theme plugin and its ThemesApi.
// isFallback is true when the configured theme is unavailable and the
// built-in core theme is used instead.
func (self *PluginsMgr) GetAdminTheme() (*PluginApi, *ThemesApi, bool, error) {
	cfg, err := config.ReadThemesConfig()
	if err != nil {
		return nil, nil, false, err
	}

	pkg := cfg.AdminThemePkg
	p, ok := self.FindByPkg(pkg)
	if ok {
		themeApi := p.Themes().(*ThemesApi)
		if themeApi.AdminTheme != nil {
			return p.(*PluginApi), themeApi, false, nil
		}
	}

	// Fall back to the built-in core theme.
	coreThemeApi := self.CoreAPI.ThemesAPI
	if coreThemeApi == nil || coreThemeApi.AdminTheme == nil {
		return nil, nil, false, fmt.Errorf("admin theme plugin '%s' is not installed and core theme is not registered", pkg)
	}
	return self.CoreAPI, coreThemeApi, true, nil
}

// GetPortalTheme returns the active portal theme plugin and its ThemesApi.
// isFallback is true when the configured theme is unavailable and the
// built-in core theme is used instead.
func (self *PluginsMgr) GetPortalTheme() (*PluginApi, *ThemesApi, bool, error) {
	cfg, err := config.ReadThemesConfig()
	if err != nil {
		return nil, nil, false, err
	}

	pkg := cfg.PortalThemePkg
	p, ok := self.FindByPkg(pkg)
	if ok {
		themeApi := p.Themes().(*ThemesApi)
		if themeApi.PortalTheme != nil {
			return p.(*PluginApi), themeApi, false, nil
		}
	}

	// Fall back to the built-in core theme.
	coreThemeApi := self.CoreAPI.ThemesAPI
	if coreThemeApi == nil || coreThemeApi.PortalTheme == nil {
		return nil, nil, false, fmt.Errorf("portal theme plugin '%s' is not installed and core theme is not registered", pkg)
	}
	return self.CoreAPI, coreThemeApi, true, nil
}

// InstallFromStore downloads the plugin zip from the marketplace, installs it
// using an encrypted scratch disk, and registers it live without a restart.
// zipURL is the transient download URL — it is not persisted on the def.
func (self *PluginsMgr) InstallFromStore(def sdkutils.PluginSrcDef, zipURL string) error {
	def.Src = sdkutils.PluginSrcStore

	info, err := plugins.InstallFromPluginStore(self.db.DB, def, zipURL, plugins.InstallOpts{
		Def:          def,
		RemoveSrc:    true,
		Encrypt:      true,
		ForceInstall: true,
	})
	if err != nil {
		return fmt.Errorf("InstallFromStore: %w", err)
	}

	installPath := plugins.GetInstallPath(info.Package)
	p := NewPluginApi(installPath, info, self.globalAssets, self, self.trfkMgr, self.wifiMgr)
	return self.RegisterPlugin(p)
}

// Uninstall marks the plugin for removal on the next restart.
func (self *PluginsMgr) Uninstall(pkg string) error {
	return plugins.MarkToRemove(pkg)
}

// IsToBeRemoved returns true if the plugin has been marked for removal.
func (self *PluginsMgr) IsToBeRemoved(pkg string) bool {
	return plugins.IsToBeRemoved(pkg)
}

// HasPendingUpdate returns true if a downloaded update is waiting to be applied.
func (self *PluginsMgr) HasPendingUpdate(pkg string) bool {
	return plugins.HasPendingUpdate(pkg)
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
