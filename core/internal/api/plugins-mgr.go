package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	sdkplugin "sdk/api"

	"core/db"
	"core/db/models"
	"core/internal/events"
	machineuid "core/internal/modules/machine-uid"
	"core/internal/modules/pluginreport"
	"core/internal/modules/scheduler"
	"core/internal/modules/ubus"
	"core/internal/network"
	"core/internal/plugindeps"
	corerpc "core/internal/rpc"
	"core/internal/rpc/rpc_flarewifi_v3"
	"core/internal/sessmgr"
	"core/utils/config"
	"core/utils/env"
	"core/utils/migrate"
	"core/utils/plugins"
	cmd "core/utils/shell"
	"core/utils/tags"

	"github.com/Masterminds/semver/v3"
	sdkutils "github.com/flarewifi/sdk-utils"
	"github.com/twitchtv/twirp"
)

// ErrPaymentRequired is returned by install paths when a paid plugin is being
// installed on a machine that is not purchased to it. Callers (e.g. the store UI)
// can detect it with errors.Is to show "purchase required" instead of a failure.
// It aliases the SDK sentinel so errors.Is matches across the plugin boundary.
var ErrPaymentRequired = sdkplugin.ErrPaymentRequired

// ErrPluginDisabled is returned by resolve/install paths when a plugin has been
// disabled by its developer in the cloud store — withdrawn and no longer
// installable or updatable, regardless of price or prior purchase. It is distinct
// from ErrPaymentRequired so the software-update dialog reports "plugin disabled"
// instead of a misleading "payment required" (the cloud signals it via the resolve
// response's disabled flag; see fetchStoreRelease).
var ErrPluginDisabled = errors.New("plugin disabled")

// ErrPluginBlocked is returned by on-device install paths (local/git) when the
// plugin being installed is on the superuser denylist, as answered live by the
// cloud (IsPluginBlocked) keyed on its plugin.json package/name. Distinct from
// ErrPluginDisabled (a developer withdrawal) — this is an operator block. The
// wrapped message carries the operator-supplied reason when one was given.
var ErrPluginBlocked = errors.New("plugin blocked")

func NewPluginMgr(d *db.Database, m *models.Models, paymgr *PaymentsMgr, clntReg *sessmgr.ClientRegister, clntMgr *sessmgr.SessionsMgr, trfkMgr *network.TrafficMgr, eventsMgr *events.EventsManager, schedulerMgr *scheduler.Manager) *PluginsMgr {
	pmgr := &PluginsMgr{
		db:           d,
		models:       m,
		paymgr:       paymgr,
		clntReg:      clntReg,
		clntMgr:      clntMgr,
		trfkMgr:      trfkMgr,
		eventsMgr:    eventsMgr,
		schedulerMgr: schedulerMgr,
		plugins:      []*PluginApi{},
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
	schedulerMgr *scheduler.Manager
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
func (self *PluginsMgr) InstallPlugin(def sdkutils.PluginSrcDef) (sdkplugin.IPluginInstall, error) {
	// Devkit builds may only install plugins from on-disk source under
	// data/plugins/local/ or data/plugins/devel/. Store and git installs are
	// rejected so the core never contacts the cloud. The developer panel uploads
	// land in the local directory (built once, then loaded); the devel directory
	// holds editable in-tree sources rebuilt at boot.
	if tags.IsDevkit() {
		withinLocal := pathWithinDir(def.LocalPath, sdkutils.PathPluginLocalDir)
		withinDevel := pathWithinDir(def.LocalPath, sdkutils.PathPluginDevelDir)
		if def.Src != sdkutils.PluginSrcLocal || (!withinLocal && !withinDevel) {
			return nil, fmt.Errorf("InstallPlugin: devkit builds may only install plugins from %s or %s", sdkutils.PathPluginLocalDir, sdkutils.PathPluginDevelDir)
		}
	}

	pkg := def.StorePackage
	if pkg == "" {
		pkg = def.LocalPath
	}

	// Membership enforcement is server-side now: an à-la-carte build of a plugin
	// pinned by an installed bundle is refused by the cloud (RequestPluginBuild's
	// managed-member gate), and the store UI disables the button via the server's
	// managed_by_meta signal. We deliberately do NOT pre-check config/plugins.json
	// here — the cloud is the single source of truth for bundle membership, so a
	// machine with a tampered config cannot smuggle a member install past it.

	// Validate payment up front for store installs so the caller gets a synchronous
	// ErrPaymentRequired (and can redirect to checkout) instead of only discovering
	// it later via the background handle. fetchStoreRelease still re-checks during
	// the install, and the cloud withholds the download, so this is not the only gate.
	if def.Src == sdkutils.PluginSrcStore {
		info, err := self.CheckPurchase(def.StorePackage)
		if err != nil {
			return nil, fmt.Errorf("InstallPlugin: validate purchase for %q: %w", def.StorePackage, err)
		}
		// Availability is gated before payment: a plugin that can't be installed at
		// all (currently: withdrawn/disabled by its developer) must report that rather
		// than a misleading "payment required" (mirrors fetchStoreRelease's order).
		if !info.Available {
			return nil, fmt.Errorf("%w: %q", ErrPluginDisabled, def.StorePackage)
		}
		if info.RequiresPayment() {
			return nil, fmt.Errorf("%w: %q", ErrPaymentRequired, def.StorePackage)
		}
	}

	h := newPluginInstall(pkg)
	// The install runs in the background so the caller can stream Progress() while
	// it proceeds; finish() records the result and closes the handle exactly once.
	go func() {
		var requiresReboot bool
		err := self.runInstall(def, h.emit, &requiresReboot)
		h.requiresReboot = requiresReboot
		h.finish(err)
	}()
	return h, nil
}

// checkPluginBlocked asks the cloud whether a plugin (identified by its plugin.json
// package/name) is on the superuser denylist, returning ErrPluginBlocked (wrapping
// the operator-supplied reason, when given) if so. It is the install-time gate for
// on-device installs, wired in as the InstallOpts.PreBuild hook so a blocked plugin
// is refused BEFORE it is compiled on-device. Its signature matches PreBuild.
//
// Fail-open by design: a missing machine identity or any cloud/transport error
// leaves the install to proceed. This mirrors the daily denylist reconcile
// (jobs.reconcileBlockedPlugins), which never blocks on a failed fetch — a cloud
// outage must not break installs, and the marker-based boot loader still catches a
// blocked plugin on the next reboot. The recover() contains the shared RPC helper,
// which panics on a header error and would otherwise crash the install goroutine.
func (self *PluginsMgr) checkPluginBlocked(info sdkutils.PluginInfo) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = nil
		}
	}()

	_, machineID := machineuid.GetMachineUID()
	if machineID == "" {
		return nil
	}

	srv, ctx := corerpc.GetTwirpServiceAndCtx()
	resp, rpcErr := srv.IsPluginBlocked(ctx, &rpc_flarewifi_v3.IsPluginBlockedRequest{
		MachineId: machineID,
		Package:   info.Package,
		Name:      info.Name,
	})
	if rpcErr != nil {
		return nil
	}
	if resp.GetBlocked() {
		if reason := strings.TrimSpace(resp.GetReason()); reason != "" {
			return fmt.Errorf("%w: %s", ErrPluginBlocked, reason)
		}
		return ErrPluginBlocked
	}
	return nil
}

// runInstall performs a plugin install synchronously, reporting stage progress
// through emit (never nil here — the InstallPlugin handle supplies it). The
// percentages are coarse, monotonic checkpoints: the cloud build only reports
// queued/processing/done, so the build phase ramp inside fetchPrebuiltPluginURL is
// an estimate. requiresReboot is set to true when the caller must reboot the
// machine before the install/update takes effect (see the store-plugin
// already-loaded check below); it is never read by this method, only written.
func (self *PluginsMgr) runInstall(def sdkutils.PluginSrcDef, emit progressEmitter, requiresReboot *bool) error {
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
		rel, err = self.fetchStoreRelease(def.StorePackage, def.StorePluginVersion, "")
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
		tarballURL, err = self.fetchPrebuiltPluginURL(def.StorePackage, rel.Version, "", emit, "")
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
			PreBuild:     self.checkPluginBlocked,
		})

	case sdkutils.PluginSrcLocal:
		emit.call(sdkplugin.PluginInstallStageBuilding, 30, "")
		info, err = plugins.InstallFromLocalPath(self.db.DB, def, plugins.InstallOpts{
			Def:          def,
			ForceInstall: true,
			PinnedDeps:   pinned,
			PreBuild:     self.checkPluginBlocked,
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

		// An already-loaded standalone store plugin's old .so is mapped into this
		// process; Go's plugin.Open cannot hot-reload it (same constraint as
		// installStoreMember's register=false path for meta members). The files
		// overwritten above only take effect on the next reboot, so skip live
		// re-registration here — retrying it would get Go's cached (stale) plugin
		// AND double-track the package in self.plugins, since the old entry is
		// still there. Report the reboot requirement to the caller instead of
		// rebooting automatically: unlike a meta-member update or a local-plugin
		// recompile conflict, this is a single plugin the admin explicitly chose to
		// update, so the store UI asks before rebooting rather than doing it silently.
		if _, alreadyLoaded := self.FindByPkg(info.Package); alreadyLoaded {
			*requiresReboot = true
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
//
// installingMeta names the bundle currently being expanded ("" for a normal
// à-la-carte fetch). It is forwarded to the cloud, which validates that pkg is a real
// member of that bundle and resolves the member's coverage against it — so a paid
// member is covered even during the bundle's first install, before the machine reports
// the meta as installed. The machine no longer reports its full installed-meta list;
// the cloud derives that from server state (machine_plugins).
func (self *PluginsMgr) fetchStoreRelease(pkg string, version string, installingMeta string) (storeRelease, error) {
	if pkg == "" {
		return storeRelease{}, errors.New("store package is required")
	}

	srv, ctx := corerpc.GetTwirpServiceAndCtx()
	resp, err := srv.FetchLatestPluginReleaseByPackage(ctx, &rpc_flarewifi_v3.FetchLatestPluginReleaseByPackageRequest{
		PluginPackage:  pkg,
		Version:        version,
		InstallingMeta: installingMeta,
	})
	if err != nil {
		// The cloud signals operator-safe rejections (off-channel version, version
		// not found) with deliberate twirp codes the transport layer never emits.
		// Translate those into a StoreReleaseError carrying the curated reason so the
		// store UI can show it; any other error (incl. transport errors that would
		// leak the endpoint URL) stays wrapped and only reaches the logs.
		if reason := operatorSafeStoreReason(err); reason != "" {
			return storeRelease{}, fmt.Errorf("fetch release %q for %q: %w", version, pkg, &sdkplugin.StoreReleaseError{Reason: reason})
		}
		return storeRelease{}, fmt.Errorf("fetch release %q for %q: %w", version, pkg, err)
	}

	// Disabled gate. The cloud sets disabled (and withholds the zip URL) for a
	// plugin its developer has withdrawn from the store. Checked BEFORE the payment
	// gate so a disabled paid plugin reports "plugin disabled" rather than a
	// misleading "payment required" (the server gates it in the same order).
	if resp.GetDisabled() {
		return storeRelease{}, fmt.Errorf("%w: %q", ErrPluginDisabled, pkg)
	}

	// Payment gate. The cloud sets requires_payment (and withholds the zip URL)
	// for a paid plugin this machine is not purchased to. This is the single choke
	// point for both standalone installs and meta-member installs, so a dropped
	// meta member that the machine wants to keep is caught here too.
	if resp.GetRequiresPayment() {
		return storeRelease{}, fmt.Errorf("%w: %q", ErrPaymentRequired, pkg)
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

// operatorSafeStoreReason returns a curated, UI-safe reason string when err is a
// store-resolve rejection the cloud raised with a deliberate twirp code, else "".
// FailedPrecondition (version published on another channel) and NotFound (version
// does not exist) are application codes the cloud sets intentionally; the twirp
// transport layer raises Internal/Unavailable for connection failures, so gating on
// these two codes guarantees we never surface a message containing the endpoint URL.
func operatorSafeStoreReason(err error) string {
	var twerr twirp.Error
	if !errors.As(err, &twerr) {
		return ""
	}
	switch twerr.Code() {
	case twirp.FailedPrecondition, twirp.NotFound:
		return twerr.Msg()
	default:
		return ""
	}
}

// CheckPurchase asks the cloud store whether this machine may install pkg and
// at what price. Auth and machine identity are handled by the shared RPC client.
func (self *PluginsMgr) CheckPurchase(pkg string) (sdkplugin.PluginPurchaseInfo, error) {
	if pkg == "" {
		return sdkplugin.PluginPurchaseInfo{}, errors.New("package is required")
	}

	srv, ctx := corerpc.GetTwirpServiceAndCtx()
	resp, err := srv.CheckPluginPurchase(ctx, &rpc_flarewifi_v3.CheckPluginPurchaseRequest{
		Package: pkg,
	})
	if err != nil {
		return sdkplugin.PluginPurchaseInfo{}, fmt.Errorf("check purchase for %q: %w", pkg, err)
	}
	// Success=false covers both an unexpected server-side failure (DB hiccup,
	// etc.) and the conclusive case of no pricing row for pkg at all (unknown
	// package) -- the server answers HTTP 200 either way (see
	// CheckPluginPurchase) since err != nil here is already reserved for a
	// plain transport failure. Surface it the same way as that transport
	// failure so callers (ValidateStorePlugins) never disable an
	// already-installed plugin based on inconclusive/unrecognized data.
	if !resp.GetSuccess() {
		return sdkplugin.PluginPurchaseInfo{}, fmt.Errorf("check purchase for %q: %s", pkg, resp.GetErrorMessage())
	}

	return sdkplugin.PluginPurchaseInfo{
		Package:              pkg,
		Purchased:            resp.GetPurchased(),
		IsFree:               resp.GetIsFree(),
		PricingType:          resp.GetPricingType(),
		SubscriptionInterval: resp.GetSubscriptionInterval(),
		PriceUsdCents:        resp.GetPriceUsdCents(),
		LocalCurrency:        resp.GetLocalCurrency(),
		LocalPriceCents:      resp.GetLocalPriceCents(),
		DisplayCurrency:      resp.GetDisplayCurrency(),
		DisplayPriceCents:    resp.GetDisplayPriceCents(),
		ExpiresAt:            resp.GetExpiresAt(),
		Reason:               resp.GetReason(),
		// A disabled plugin (withdrawn by its developer) is the current cause of
		// unavailability; map it to the general Available flag so callers gate on
		// "can this be installed at all" rather than one specific reason.
		Available: !resp.GetDisabled(),
	}, nil
}

// buildPurchaseURL is the implementation behind IPluginsMgrApi.GetPurchaseURL.
// It is called by the per-plugin wrapper (see PluginApi.PluginsMgr), which
// supplies the calling plugin (owner) so the callback route resolves in that
// plugin's namespace. See the interface doc for the full contract.
func (self *PluginsMgr) buildPurchaseURL(r *http.Request, owner *PluginApi, pkg string, callbackRouteName string, pairs ...string) (string, error) {
	if pkg == "" {
		return "", errors.New("package is required")
	}
	if callbackRouteName == "" {
		return "", errors.New("callback route name is required")
	}

	// The machine's browser-facing base URL (the LAN IP / hostname the owner used
	// to reach the store), so the cloud redirect lands back on this device rather
	// than on loopback.
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	machineBaseURL := scheme + "://" + r.Host

	// Resolve the calling plugin's callback route to a relative path. pairs fill the
	// route's path parameters exactly like UrlForRoute.
	callbackPath := owner.HttpAPI.httpRouter.UrlForRoute(sdkplugin.PluginRouteName(callbackRouteName), pairs...)
	if callbackPath == "" {
		return "", fmt.Errorf("callback route %q is not registered", callbackRouteName)
	}

	// Always carry the package so the handler knows what to install, regardless of
	// the route's own parameters. Use "&" if UrlForRoute already produced a query.
	sep := "?"
	if strings.Contains(callbackPath, "?") {
		sep = "&"
	}
	returnURL := machineBaseURL + callbackPath + sep + "pkg=" + url.QueryEscape(pkg)

	_, machineID := machineuid.GetMachineUID()

	// env.WebBaseURL() is the cloud dashboard origin (www.<SERVER_DOMAIN>) where
	// the plugin-checkout page is served.
	return env.WebBaseURL() + "/plugin-checkout" +
		"?machine_id=" + url.QueryEscape(machineID) +
		"&package=" + url.QueryEscape(pkg) +
		"&return_url=" + url.QueryEscape(returnURL), nil
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
	memberUpdated := false
	memberPkgs := make([]string, 0, len(rel.Members))

	for _, m := range rel.Members {
		memberPkgs = append(memberPkgs, m.Package)

		memberDef := sdkutils.PluginSrcDef{
			Src:                sdkutils.PluginSrcStore,
			StorePackage:       m.Package,
			StorePluginVersion: m.Version,
		}

		// Already present: adopt as-is UNLESS this bundle release pins a newer
		// version, in which case update the member to it. Without this, re-pinning a
		// bundle to a new release would bump the meta record but leave every existing
		// member stranded on its old version (the pre-fix bug).
		if existing, ok := self.FindByPkg(m.Package); ok {
			if !memberNeedsUpdate(existing.Info().Version, m.Version) {
				continue
			}
			// Update in place but do NOT re-register: the old member .so is already
			// mapped into this process and Go's plugin.Open cannot hot-reload it, so
			// re-registering would re-Init the stale cached .so. Overwriting the files
			// (ForceInstall) stages the new build; a reboot below loads it cleanly. An
			// updated member is NOT added to `installed` — a later rollback must not
			// uninstall a member that predated this bundle update.
			if err := self.installStoreMember(memberDef, def.StorePackage, false); err != nil {
				self.rollbackMeta(installed)
				return fmt.Errorf("update member %q: %w", m.Package, err)
			}
			memberUpdated = true
			continue
		}

		// Fresh member: install and register live so it works without a reboot.
		if err := self.installStoreMember(memberDef, def.StorePackage, true); err != nil {
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
	// to the lock and reboot to apply if any were recompiled. A member update also
	// needs a reboot on its own — the refreshed .so was staged, not hot-loaded — so
	// reboot when either happened.
	recompiled := self.reconcileLocalPluginsWithLock(plugindeps.Fetch(""))
	if recompiled || memberUpdated {
		self.rebootToApplyRecompiledPlugins(def.StorePackage)
	}

	return nil
}

// installStoreMember installs a single store plugin as a member of metaPkg and
// registers it live. The install is recorded as non-standalone (AsMetaMember);
// the bundle->member ownership itself is tracked by the meta record's member
// list (cfg.MetaPlugins), not in the member's own metadata.
//
// metaPkg is passed to the cloud as installing_meta on both the release fetch and the
// build request, so the cloud (a) covers a PAID member via the bundle during the
// bundle's first install — before the meta record exists on-device or in
// machine_plugins — and (b) recognizes the member build as a legitimate bundle
// expansion rather than a refused à-la-carte install of a bundle-pinned member.
//
// register controls live registration. Pass true for a fresh member so it loads and
// works without a reboot. Pass false when updating an already-loaded member: its old
// .so is already mapped into the process and Go's plugin.Open cannot hot-reload it,
// so the overwritten files are only picked up on the next reboot; re-registering here
// would re-Init the stale cached .so.
func (self *PluginsMgr) installStoreMember(def sdkutils.PluginSrcDef, metaPkg string, register bool) error {
	rel, err := self.fetchStoreRelease(def.StorePackage, def.StorePluginVersion, metaPkg)
	if err != nil {
		return err
	}
	if rel.IsMeta {
		return fmt.Errorf("meta plugin %q cannot be a member of another meta", def.StorePackage)
	}

	tarballURL, err := self.fetchPrebuiltPluginURL(def.StorePackage, rel.Version, "", nil, metaPkg)
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

	// Updating an already-loaded member: files are staged on disk, reboot applies them.
	if !register {
		return nil
	}

	installPath := plugins.GetInstallPath(info.Package)
	p := NewPluginApi(installPath, info, self.globalAssets, self, self.trfkMgr, self.wifiMgr)
	return self.RegisterPlugin(p)
}

// memberNeedsUpdate reports whether a bundle re-pin should reinstall an already-present
// member: true only when the bundle's pinned version parses as strictly newer than the
// installed one. Comparing semver (not string equality) avoids downgrading a member the
// machine already has ahead of this bundle and skips a needless reinstall when they
// match. If either version is unparseable, fall back to string inequality so a differing
// non-semver pin still triggers an update rather than being silently skipped.
func memberNeedsUpdate(installedVersion, pinnedVersion string) bool {
	cur, curErr := semver.NewVersion(installedVersion)
	next, nextErr := semver.NewVersion(pinnedVersion)
	if curErr != nil || nextErr != nil {
		return installedVersion != pinnedVersion
	}
	return next.GreaterThan(cur)
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
	err := self.uninstallPlugin(pkg)
	if err == nil {
		// Nudge the cloud to re-read installed plugins so the uninstall is reflected
		// promptly. The report excludes marked-to-remove packages, so the plugin
		// drops out of the snapshot now even though its files clear on the next reboot.
		pluginreport.ReportNowAsync()
	}
	return err
}

func (self *PluginsMgr) uninstallPlugin(pkg string) error {
	if self.isMetaPlugin(pkg) {
		return self.uninstallMeta(pkg)
	}
	if def, err := plugins.GetPluginDef(pkg); err == nil && def.Src == sdkutils.PluginSrcSystem {
		return errors.New("cannot uninstall system plugin: " + pkg)
	}
	if err := plugins.MarkToRemove(pkg); err != nil {
		return err
	}
	// The plugin's HTTP routes are already gated per-request by
	// middlewares.PluginValidityCheck (via IsToBeRemoved); stop its background
	// work too, immediately, instead of leaving it running until the next
	// reboot physically removes it. Mirrors the BlockPlugin/DisablePlugin call
	// sites in blocked-plugins.go / boot.ValidateStorePlugins. The meta cascade
	// (uninstallMeta) routes each member back through this same function, so a
	// meta-bundle uninstall cancels every member's tasks too.
	self.schedulerMgr.CancelOwner(pkg)
	return nil
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

// pathWithinDir reports whether localPath resolves to a location inside dir
// (or dir itself). Used by the devkit install guard to confine local installs to
// data/plugins/devel/. A path that cannot be resolved is treated as outside.
func pathWithinDir(localPath, dir string) bool {
	abs, err := filepath.Abs(localPath)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(dir, abs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
