package api

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	sdkplugin "sdk/api"

	"core/db"
	"core/db/models"
	"core/internal/events"
	"core/internal/modules/ubus"
	"core/internal/network"
	"core/internal/plugindeps"
	corerpc "core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v2"
	"core/internal/sessmgr"
	"core/utils/config"
	"core/utils/migrate"
	"core/utils/plugins"
	cmd "core/utils/shell"

	sdkutils "github.com/flarewifi/sdk-utils"
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

// PluginApis returns the live, concretely-typed plugin list for core-internal use
// (callers that need *PluginApi rather than the IPluginApi interface). The public,
// interface-typed accessor is Plugins().
func (self *PluginsMgr) PluginApis() []*PluginApi {
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

func (self *PluginsMgr) Plugins() []sdkplugin.IPluginApi {
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

// InstallPlugin installs a plugin from any source and registers it live without
// a restart. The source is selected by def.Src:
//   - store:           resolves the release via the core RPC service (using
//     def.StorePackage / def.StorePluginVersion), then downloads the
//     server-built install-ready tarball — store plugins are compiled in the
//     cloud against this machine's exact core version and platform, so nothing
//     is compiled on the device. The resolved URL is transient and not persisted.
//   - git:             clones def.GitURL at def.GitRef into a temp dir.
//   - local:           installs from def.LocalPath, a source folder already on
//     disk; the source files are left in place (RemoveSrc is forced off).
//
// ForceInstall routes the build straight to plugins/installed instead of staging
// it as a pending update.
func (self *PluginsMgr) InstallPlugin(def sdkutils.PluginSrcDef) sdkplugin.IPluginInstall {
	pkg := def.StorePackage
	if pkg == "" {
		pkg = def.LocalPath
	}
	h := newPluginInstall(pkg)
	// The install runs in the background so the caller can stream Progress() while
	// it proceeds; finish() records the result and closes the handle exactly once.
	go func() {
		h.finish(self.runInstall(def, h.emit))
	}()
	return h
}

// runInstall performs a plugin install synchronously, reporting stage progress
// through emit (never nil here — the InstallPlugin handle supplies it). The
// percentages are coarse, monotonic checkpoints: the cloud build only reports
// queued/processing/done, so the build phase ramp inside fetchPrebuiltPluginURL is
// an estimate.
func (self *PluginsMgr) runInstall(def sdkutils.PluginSrcDef, emit progressEmitter) error {
	var info sdkutils.PluginInfo
	var err error

	// Pin the on-device build to the running core's dependency lock so the compiled
	// .so is ABI-compatible with the core and other installed plugins. Empty lock or
	// unreachable cloud => nil => unpinned build (graceful, see plugindeps.Fetch).
	pinned := plugindeps.Fetch("")

	switch def.Src {
	case sdkutils.PluginSrcStore:
		emit.call(sdkplugin.PluginInstallStageResolving, 5, "")
		var rel storeRelease
		rel, err = self.fetchStoreRelease(def.StorePackage, def.StorePluginVersion)
		if err != nil {
			return fmt.Errorf("InstallPlugin: %w", err)
		}
		// A meta bundle has no zip of its own — install each of its members and
		// record the bundle. installStoreMeta registers members live itself.
		if rel.IsMeta {
			emit.call(sdkplugin.PluginInstallStageInstalling, 30, "")
			return self.installStoreMeta(def, rel)
		}
		// rel.Version is the concrete semver the lookup resolved (the prebuild
		// RPC rejects an empty version when def.StorePluginVersion was "latest").
		var tarballURL string
		tarballURL, err = self.fetchPrebuiltPluginURL(def.StorePackage, rel.Version, "", emit)
		if err != nil {
			return fmt.Errorf("InstallPlugin: %w", err)
		}
		emit.call(sdkplugin.PluginInstallStageInstalling, 85, "")
		info, err = plugins.InstallFromPrebuilt(self.db.DB, tarballURL, plugins.InstallOpts{
			Def:          def,
			ForceInstall: true,
		})

	case sdkutils.PluginSrcGit:
		// Git/local plugins compile on-device; there is no cloud build to poll, so a
		// single building checkpoint covers the (potentially long) compile.
		emit.call(sdkplugin.PluginInstallStageBuilding, 30, "")
		info, err = plugins.InstallFromGitSrc(self.db.DB, def, plugins.InstallOpts{
			Def:          def,
			RemoveSrc:    true,
			ForceInstall: true,
			PinnedDeps:   pinned,
		})

	case sdkutils.PluginSrcLocal:
		emit.call(sdkplugin.PluginInstallStageBuilding, 30, "")
		info, err = plugins.InstallFromLocalPath(self.db.DB, def, plugins.InstallOpts{
			Def:          def,
			ForceInstall: true,
			PinnedDeps:   pinned,
		})

	default:
		return fmt.Errorf("InstallPlugin: invalid plugin source: %q", def.Src)
	}

	if err != nil {
		return fmt.Errorf("InstallPlugin: %w", err)
	}

	emit.call(sdkplugin.PluginInstallStageInstalling, 90, "")

	// A store plugin's .so is pinned to the (store-prioritized) lock. A local plugin
	// built against an older lock can share a module at a different version, making
	// the two .so files mutually ABI-incompatible — the new store plugin would fail
	// to load alongside it. Recompile any such local plugin pinned to the lock; if
	// any was recompiled we must reboot to load the coherent set (a running .so can't
	// be hot-reloaded), so we skip the live registration below.
	if def.Src == sdkutils.PluginSrcStore {
		if recompiled := self.reconcileLocalPluginsWithLock(pinned); recompiled {
			self.rebootToApplyRecompiledPlugins(info.Package)
			return nil
		}
	}

	installPath := plugins.GetInstallPath(info.Package)
	p := NewPluginApi(installPath, info, self.globalAssets, self, self.trfkMgr, self.wifiMgr)
	return self.RegisterPlugin(p)
}

// reconcileLocalPluginsWithLock recompiles every installed LOCAL plugin whose current
// build disagrees with the dependency lock (a shared module at a different
// version/hash), pinning it to the lock so it is ABI-compatible with the core and the
// just-installed store plugin.
//
// The conformance direction is forced by HOW each kind of plugin is built: a store
// plugin is compiled ONCE in the cloud and cached as an immutable artifact shared by
// every machine — it cannot be recompiled here to match one device's local plugins.
// A local plugin is compiled ON-DEVICE and can be rebuilt cheaply. So the (cached)
// store/core versions in the lock are authoritative and local plugins bend to them —
// never the reverse. Returns whether any plugin was recompiled. Best-effort per
// plugin: a single failure is logged and the sweep continues, so one bad local plugin
// never aborts the store install.
func (self *PluginsMgr) reconcileLocalPluginsWithLock(lock []plugins.LockedGoModule) bool {
	if len(lock) == 0 {
		return false
	}

	srcDirs, err := plugins.InstalledLocalPluginSrcDirs()
	if err != nil {
		self.CoreAPI.LoggerAPI.Error(fmt.Sprintf("reconcile: enumerate local plugins: %v", err))
		return false
	}

	recompiled := false
	for _, srcDir := range srcDirs {
		needs, err := plugins.LocalPluginNeedsRepin(srcDir, lock)
		if err != nil {
			self.CoreAPI.LoggerAPI.Error(fmt.Sprintf("reconcile: check %s: %v", sdkutils.StripRootPath(srcDir), err))
			continue
		}
		if !needs {
			continue
		}
		self.CoreAPI.LoggerAPI.Info(fmt.Sprintf("reconcile: recompiling local plugin %s against the dependency lock (store deps prioritized)", sdkutils.StripRootPath(srcDir)))
		def := sdkutils.PluginSrcDef{Src: sdkutils.PluginSrcLocal, LocalPath: sdkutils.StripRootPath(srcDir)}
		if _, err := plugins.InstallFromLocalPath(self.db.DB, def, plugins.InstallOpts{
			Def:          def,
			ForceInstall: true,
			PinnedDeps:   lock,
		}); err != nil {
			self.CoreAPI.LoggerAPI.Error(fmt.Sprintf("reconcile: recompile %s: %v", sdkutils.StripRootPath(srcDir), err))
			continue
		}
		recompiled = true
	}
	return recompiled
}

// rebootToApplyRecompiledPlugins reboots so the core, the freshly-recompiled local
// plugins, and the new store plugin all load together against one coherent dependency
// set. Go's plugin.Open cannot reload a .so already mapped into the running process,
// so the recompiled local .so files only take effect on a fresh start. Dev shell.Exec
// ignores reboot. Mirrors adminctrl/power-ctrl.go's reboot pattern.
func (self *PluginsMgr) rebootToApplyRecompiledPlugins(pkg string) {
	self.CoreAPI.LoggerAPI.Info(fmt.Sprintf("installed store plugin %s and recompiled conflicting local plugin(s); rebooting to apply", pkg))
	go func() {
		time.Sleep(3 * time.Second)
		cmd.Exec("reboot", nil)
	}()
}

// storeRelease is the resolved store lookup for a package: either a single
// downloadable release (ZipURL set) or a meta bundle (IsMeta with Members).
type storeRelease struct {
	ZipURL  string
	IsMeta  bool
	Name    string
	Version string
	Members []storeMember
}

// storeMember is a meta bundle member pinned to a specific version.
type storeMember struct {
	Package string
	Version string
}

// fetchStoreRelease resolves a store package via the core RPC service. For a
// normal plugin it returns the download (zip) URL for the requested version (or
// latest when version is empty). For a meta bundle it returns IsMeta with the
// pinned member list. Auth and machine identity are handled by the shared RPC
// client; any resolved URL is transient and never persisted on the source def.
func (self *PluginsMgr) fetchStoreRelease(pkg string, version string) (storeRelease, error) {
	if pkg == "" {
		return storeRelease{}, errors.New("store package is required")
	}

	srv, ctx := corerpc.GetTwirpServiceAndCtx()
	resp, err := srv.FetchLatestPluginReleaseByPackage(ctx, &rpc_flarewifi_v2.FetchLatestPluginReleaseByPackageRequest{
		PluginPackage: pkg,
		Version:       version,
	})
	if err != nil {
		return storeRelease{}, fmt.Errorf("fetch release %q for %q: %w", version, pkg, err)
	}

	rel := storeRelease{
		IsMeta:  resp.GetIsMeta(),
		Name:    resp.GetName(),
		Version: resp.GetVersion(),
	}

	if rel.IsMeta {
		for _, m := range resp.GetMembers() {
			rel.Members = append(rel.Members, storeMember{Package: m.GetPackage(), Version: m.GetVersion()})
		}
		return rel, nil
	}

	rel.ZipURL = resp.GetPluginRelease().GetZipFileUrl()
	if rel.ZipURL == "" {
		return storeRelease{}, fmt.Errorf("no download url for plugin %q", pkg)
	}

	return rel, nil
}

// installStoreMeta installs every member of a store meta bundle and records the
// bundle. Members already present are adopted (ownership is recorded by the saved
// meta record); missing members are installed fresh at their pinned version. On
// any failure, members installed in this call are marked for removal so the
// operation leaves no partial bundle behind.
func (self *PluginsMgr) installStoreMeta(def sdkutils.PluginSrcDef, rel storeRelease) error {
	if len(rel.Members) == 0 {
		return fmt.Errorf("meta plugin %q has no members", def.StorePackage)
	}

	var installed []string
	memberPkgs := make([]string, 0, len(rel.Members))

	for _, m := range rel.Members {
		memberPkgs = append(memberPkgs, m.Package)

		// Already present — adopt it; ownership is recorded by the meta record.
		if _, ok := self.FindByPkg(m.Package); ok {
			continue
		}

		memberDef := sdkutils.PluginSrcDef{
			Src:                sdkutils.PluginSrcStore,
			StorePackage:       m.Package,
			StorePluginVersion: m.Version,
		}
		if err := self.installStoreMember(memberDef, def.StorePackage); err != nil {
			self.rollbackMeta(installed)
			return fmt.Errorf("install member %q: %w", m.Package, err)
		}
		installed = append(installed, m.Package)
	}

	rec := sdkutils.MetaPlugin{
		Package: def.StorePackage,
		Name:    rel.Name,
		Version: rel.Version,
		Members: memberPkgs,
	}
	if err := plugins.WriteMetaRecord(rec); err != nil {
		self.rollbackMeta(installed)
		return fmt.Errorf("save meta record for %q: %w", def.StorePackage, err)
	}

	// Same as a single store install: a member may have bumped a shared module in the
	// lock, leaving a local plugin stale. Recompile conflicting local plugins pinned
	// to the lock and reboot to apply if any were recompiled.
	if recompiled := self.reconcileLocalPluginsWithLock(plugindeps.Fetch("")); recompiled {
		self.rebootToApplyRecompiledPlugins(def.StorePackage)
	}

	return nil
}

// installStoreMember installs a single store plugin as a member of metaPkg and
// registers it live. The install is recorded as non-standalone (AsMetaMember);
// the bundle->member ownership itself is tracked by the meta record's member
// list (cfg.MetaPlugins), not in the member's own metadata.
func (self *PluginsMgr) installStoreMember(def sdkutils.PluginSrcDef, metaPkg string) error {
	rel, err := self.fetchStoreRelease(def.StorePackage, def.StorePluginVersion)
	if err != nil {
		return err
	}
	if rel.IsMeta {
		return fmt.Errorf("meta plugin %q cannot be a member of another meta", def.StorePackage)
	}

	tarballURL, err := self.fetchPrebuiltPluginURL(def.StorePackage, rel.Version, "", nil)
	if err != nil {
		return err
	}

	info, err := plugins.InstallFromPrebuilt(self.db.DB, tarballURL, plugins.InstallOpts{
		Def:          def,
		ForceInstall: true,
		AsMetaMember: metaPkg,
	})
	if err != nil {
		return err
	}

	installPath := plugins.GetInstallPath(info.Package)
	p := NewPluginApi(installPath, info, self.globalAssets, self, self.trfkMgr, self.wifiMgr)
	return self.RegisterPlugin(p)
}

// rollbackMeta marks freshly-installed meta members for removal after a failed
// bundle install. Removal is applied on the next restart.
func (self *PluginsMgr) rollbackMeta(installed []string) {
	for _, pkg := range installed {
		if err := self.UninstallPlugin(pkg); err != nil {
			self.CoreAPI.Logger().Error(fmt.Sprintf("rollbackMeta: uninstall %s: %v", pkg, err))
		}
	}
}

// UninstallPlugin removes a plugin or meta bundle through a single entry point. A
// meta bundle has no plugin.so of its own, so it is routed to the meta cascade
// (remove the record and mark orphaned members for removal). Otherwise the plugin
// is marked for removal on the next restart. System plugins ship with the product
// image and are foundational, so they can never be uninstalled — this is the last
// line of defense, enforced regardless of which caller (UI, RPC, meta rollback)
// reaches it.
func (self *PluginsMgr) UninstallPlugin(pkg string) error {
	if self.isMetaPlugin(pkg) {
		return self.uninstallMeta(pkg)
	}
	if def, err := plugins.GetPluginDef(pkg); err == nil && def.Src == sdkutils.PluginSrcSystem {
		return errors.New("cannot uninstall system plugin: " + pkg)
	}
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

// SourceDef returns the source definition for an installed plugin (where it
// came from: git, store, system, local, or zip). Returns (zero-value, false)
// when the package is not installed or the metadata cannot be read.
func (self *PluginsMgr) SourceDef(pkg string) (sdkutils.PluginSrcDef, bool) {
	def, err := plugins.GetPluginDef(pkg)
	if err != nil {
		return sdkutils.PluginSrcDef{}, false
	}
	return def, true
}

// RerunPluginMigrations re-runs migrations for all loaded plugins
// This is used after database reset to recreate plugin tables
func (self *PluginsMgr) RerunPluginMigrations(newDB *sql.DB) error {
	for _, p := range self.plugins {
		// Skip core plugin (already migrated)
		if p.Info().Package == "com.flarego.core" {
			continue
		}

		pluginDir := p.Dir()
		migdir := filepath.Join(pluginDir, "resources/migrations")

		if !sdkutils.FsExists(migdir) {
			continue
		}

		// Pass the plugin directory, not the migrations subdirectory
		// This ensures proper temp directory isolation per plugin
		if err := migrate.MigrateUp(newDB, pluginDir); err != nil {
			return fmt.Errorf("failed to run migrations for plugin %s: %w", p.Info().Package, err)
		}
	}

	return nil
}
