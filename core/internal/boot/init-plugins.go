//go:build !mono

package boot

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"core/internal/api"
	"core/utils/env"
	"core/utils/migrate"
	"core/utils/plugins"
	"core/utils/tags"

	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// loaderEmitsPluginProgress reports whether InitPlugins itself publishes the
// per-plugin booting-page checklist. In the non-mono build, plugins are mapped
// from .so one at a time here — the visibly slow phase — so the loader emits the
// checklist during load and InitLoadedPlugins must NOT re-emit it (mono's loader
// is instant, so it defers to InitLoadedPlugins; see the mono variant).
const loaderEmitsPluginProgress = true

func InitPlugins(g *api.CoreGlobals) error {
	db := g.CoreAPI.SqlDB()

	// Make plugins.json reflect what is physically under data/plugins/{local,devel} —
	// the authoritative location for local/devel plugin sources — before anything
	// reads the config this boot. A developer can drop a plugin into
	// data/plugins/devel/ (or delete one) and have it registered/unregistered on the
	// next boot without hand-editing plugins.json. Best-effort: a reconcile failure is
	// logged but never aborts boot, since the compile/load phases scan the dirs
	// directly and don't depend on this.
	if added, removed, err := plugins.ReconcileLocalDevelPluginsConfig(); err != nil {
		g.CoreAPI.Logger().Error("boot: reconcile plugins.json failed: " + err.Error())
	} else if len(added) > 0 || len(removed) > 0 {
		g.CoreAPI.Logger().Info(fmt.Sprintf("boot: plugins.json reconciled (added %v, removed %v)", added, removed))
	}

	localPlugins := plugins.LocalPluginSrcDefs()
	systemPlugins := plugins.SystemPluginSrcDefs()
	develPlugins := plugins.DevelPluginSrcDefs()

	// Devel plugins are a development-only convenience: editable source under
	// data/plugins/devel/ that is rebuilt from scratch on every boot. They are
	// compiled and loaded ONLY in dev and devkit builds (both run as ENV_DEV).
	// Staging and production ignore the devel directory entirely — a deployed
	// device ships and loads only its prebuilt local/system plugins.
	//
	// plugins.DevelPluginSrcDefs() is the authoritative gate and already returns
	// an empty list outside dev/devkit (see its doc comment), so this is normally
	// a no-op. It is kept as a defensive reaffirmation at the boot path: a
	// development-only directory must never load on a deployed device, even if the
	// chokepoint's behavior changes.
	if !env.IsDevEnv() {
		develPlugins = nil
	}

	// Compile/install phase. In dev this rebuilds each plugin's source into a .so,
	// which is the genuinely slow part of bringing plugins up (the .so mapping in
	// the load loop below is comparatively quick). Surface it as its own booting-page
	// phase with a per-plugin checklist so the long wait isn't a blank "Loading
	// plugins". System/local defs are dev-only; devel defs are always installed and
	// abort boot on failure only outside production (historical best-effort there).
	g.BootProgress.Advance(g.CoreAPI.Translate("info", "Compiling plugins"))

	// Production ships prebuilt .so files and never recompiles at boot; dev and
	// staging recompile from source so an edited plugin comes up without a manual
	// build. A compile failure aborts the boot ONLY in dev — on a deployed staging
	// device an abort is a non-zero exit that makes start.sh roll the whole software
	// update back (reverting a good update over one un-rebuildable plugin), so
	// staging tolerates it and keeps booting with that plugin absent (see env.IsDevEnv).
	abortOnCompileErr := env.IsDevEnv()
	if env.GO_ENV != env.ENV_PRODUCTION {
		if err := compilePluginDefs(g, db, systemPlugins, abortOnCompileErr); err != nil {
			return err
		}
		// Devkit, like production, ships/installs prebuilt local .so files and must
		// NOT recompile them at boot: an uploaded plugin is built once at install
		// time (the developer panel's InstallPlugin call) and thereafter only
		// loaded. Only a pure dev/sandbox/staging build rebuilds local source on
		// every boot so an edited in-tree plugin comes up without a manual build.
		if !tags.IsDevkit() {
			if err := compilePluginDefs(g, db, localPlugins, abortOnCompileErr); err != nil {
				return err
			}
		}
	}

	// develPlugins is nil outside dev/devkit (gated above), so this is a no-op on
	// staging/production.
	if err := compilePluginDefs(g, db, develPlugins, abortOnCompileErr); err != nil {
		return err
	}

	// Process pending removals before loading plugins
	for _, dir := range plugins.InstalledPluginDirs() {
		pkg := filepath.Base(dir)
		if plugins.IsToBeRemoved(pkg) {
			if err := plugins.UninstallPlugin(pkg, db); err != nil {
			}
		}
	}

	// NOTE: staged updates (core + plugins) are applied by the non-mono boot
	// script (start.sh) BEFORE the server starts — it overlays every
	// package staged under data/storage/system/updates/{pkg} onto its install
	// location. By the time we reach here the swap is already done and the new
	// plugin.so is on disk, so there is no Go-side apply step. (A Go plugin.so is
	// ABI-locked to its core build and plugin.Open cannot reload, which is exactly
	// why the swap must happen in the shell before any plugin is loaded.)

	// Register statically-linked system plugins first (generated by
	// core/cmd/sysplugin-prepare into core/internal/api/system-plugins-init.go).
	// These plugins have their Go code compiled into the core binary, so they
	// are NOT loaded via plugin.Open in the loop below — that would try to
	// dlopen a plugin.so file that does not exist for them.
	g.PluginMgr.LoadSystemPlugins(g)

	// Load (map the .so) phase — distinct from the compile phase above. Each plugin
	// is checked off as it is mapped.
	g.BootProgress.Advance(g.CoreAPI.Translate("info", "Loading plugins"))

	// Load plugins
	pluginDirs := plugins.InstalledPluginDirs()
	for _, dir := range pluginDirs {
		info, err := sdkutils.GetPluginInfoFromPath(dir)
		if err != nil {

			pkg := filepath.Base(dir)

			// In development every plugin must load successfully: abort boot so
			// the failure is fixed instead of silently shipping a broken plugin
			// set. Every deployed device (staging, sandbox, production) stays
			// resilient (notify + restore from backup + keep booting) so one bad
			// plugin can't brick a device — or revert a good update — in the field.
			if env.IsDevEnv() {
				return fmt.Errorf("plugin %q failed to load: %w", pkg, err)
			}

			notifyPluginLoadFailure(g, pkg, err)
			if err := LoadFromBackup(g, pkg); err != nil {
				notifyPluginBackupFailure(g, pkg, err)
			}

			continue
		}

		// Skip plugins withheld by ValidateStorePlugins: a store plugin whose
		// purchase has lapsed is left on disk but not loaded (the operator was
		// notified). The marker is cleared automatically once the purchase is
		// reconfirmed online, so the plugin loads again on a later boot.
		if plugins.IsDisabled(info.Package) {
			continue
		}

		// Skip plugins flagged by the cloud denylist (FetchBlockedPlugins,
		// reconciled daily by the blocked-plugins job). Files are kept on disk;
		// the marker is cleared automatically once the plugin drops off the
		// denylist, so it loads again on a later boot. Distinct from IsDisabled
		// so a block never resurrects an operator-disabled plugin and vice versa.
		if plugins.IsBlocked(info.Package) {
			continue
		}

		// Skip a plugin that was SKIPPED during a software update (its build failed and
		// the operator chose to continue): its .so is ABI-locked to the PREVIOUS core
		// and would fail plugin.Open against this one. Outside production that load
		// failure aborts boot and start.sh rolls the whole update back; skipping the
		// stale .so lets the new core boot cleanly with the plugin absent until a later
		// update rebuilds it (which clears the marker by replacing the install dir).
		if plugins.IsUpdateSkipped(info.Package) {
			continue
		}

		// Run migrations for every installed plugin dir up-front, before the
		// loader-specific paths diverge. This guarantees system plugins (whose
		// Go code is statically linked and registered by LoadSystemPlugins
		// above) still get their schema applied even if the generated loader
		// somehow doesn't run them — e.g. a binary built without the
		// sysplugin-prepare step, where LoadSystemPlugins is the no-op stub
		// and would otherwise leave migrations unrun in production
		// (env.ENV_PRODUCTION skips the dev-only InstallSrcDef block above).
		// MigrateUp is idempotent — it tracks done state per file in a tx —
		// so running it here is safe even when LoadSystemPlugins also ran it.
		// Pass the plugin root dir; MigrateUp resolves resources/migrations
		// itself (same convention as the mono loader and RunCoreMigrations).
		if err := migrate.MigrateUp(db, dir); err != nil {
		}

		// Skip any plugin already registered by LoadSystemPlugins above
		// (statically-linked system plugins compiled into the core binary).
		// Falling through would invoke plugin.Open on a plugin.so that does
		// not exist for them. Using PluginMgr.FindByPkg keeps this in lock-
		// step with whatever LoadSystemPlugins actually registered — no
		// parallel bookkeeping needed.
		if _, alreadyLoaded := g.PluginMgr.FindByPkg(info.Package); alreadyLoaded {
			continue
		}

		// Surface each plugin on the booting page as it is loaded — this map-the-.so
		// step is the visibly slow part of "Loading plugins" (Init, which runs later
		// in InitLoadedPlugins, is comparatively instant). The name is a plugin-
		// supplied display string, not translatable, so it carries no translation key.
		g.BootProgress.Substep(pluginDisplayName(info))

		p := api.NewPluginApi(dir, info, g.GlobalAssets, g.PluginMgr, g.TrafficMgr, g.WifiMgr)
		// Load (map the .so + resolve Init) but do NOT run Init yet. Init is
		// deferred to InitLoadedPlugins (offline-safe plugins) or the online
		// monitor's provisioning pass (plugins whose system_packages/preinstall
		// must run first, which needs internet). This still surfaces a stale/ABI-
		// broken .so here at boot, so the dev/prod error handling below is intact.
		err = g.PluginMgr.LoadPlugin(p)
		if err != nil {
			// Dev: a plugin that fails to compile/load aborts boot (see above).
			// Deployed devices (staging, sandbox, production): notify the admin and
			// try to recover from backup, then keep booting.
			if env.IsDevEnv() {
				return fmt.Errorf("plugin %q failed to load: %w", info.Package, err)
			}

			notifyPluginLoadFailure(g, info.Package, err)
			if err := LoadFromBackup(g, info.Package); err != nil {
				notifyPluginBackupFailure(g, info.Package, err)
			}
		}
	}

	return nil
}

// compilePluginDefs installs/compiles each src def, checking it off on the booting
// page first so the (dev) rebuild of each plugin is visible under "Compiling
// plugins". A def already flagged for removal is skipped silently. abortOnErr
// turns an install failure into a boot-aborting error (system/local + non-prod
// devel); otherwise the failure is tolerated (production devel, best-effort).
func compilePluginDefs(g *api.CoreGlobals, db *sql.DB, defs []sdkutils.PluginSrcDef, abortOnErr bool) error {
	for _, def := range defs {
		info, infoErr := sdkutils.GetPluginInfoFromPath(def.LocalPath)
		if infoErr == nil && plugins.IsToBeRemoved(info.Package) {
			continue
		}
		// A local plugin skipped during a software update (build failed, operator
		// continued) must not be recompiled here either: its source still fails to
		// build against the new core, and with abortOnErr this would abort the whole
		// boot — exactly the revert this marker prevents. Leave it skipped until a
		// later update rebuilds it.
		if infoErr == nil && plugins.IsUpdateSkipped(info.Package) {
			continue
		}
		g.BootProgress.Substep(compileDisplayName(info, infoErr, def.LocalPath))
		if _, err := plugins.InstallSrcDef(db, def, plugins.InstallOpts{ForceInstall: true, Def: def}); err != nil {
			if abortOnErr {
				return fmt.Errorf("error installing plugin %s: %w", def.LocalPath, err)
			}
		}
	}
	return nil
}

// compileDisplayName is the booting-page label for a plugin being compiled: its
// declared Name when readable, else the source directory's base name (the plugin
// info may not parse yet at compile time, so pluginDisplayName isn't enough).
func compileDisplayName(info sdkutils.PluginInfo, infoErr error, localPath string) string {
	if infoErr == nil && info.Name != "" {
		return info.Name
	}
	return filepath.Base(localPath)
}

// notifyPluginLoadFailure records a plugin load/compile failure surfaced at boot
// (e.g. a stale plugin.so that fails plugin.Open with "different version of
// package", or any other dlopen error). The detailed cause is always logged for
// diagnostics; in production it additionally raises an admin notification so the
// operator knows a plugin failed and needs attention. The user-facing message is
// kept generic — it names only the plugin package, never the underlying error or
// .so path — per the error-message hygiene rules; the specifics stay in the log.
func notifyPluginLoadFailure(g *api.CoreGlobals, pkg string, cause error) {
	if err := g.CoreAPI.Logger().Error(fmt.Sprintf("plugin %q failed to load: %v", pkg, cause)); err != nil {
	}

	// Dev surfaces failures in the console/logs and aborts boot, so no admin
	// notification is raised. Every deployed device (staging, sandbox, production)
	// keeps booting, so it must tell the operator which plugin is unavailable.
	if env.IsDevEnv() {
		return
	}

	subject := g.CoreAPI.Translate("error", "Plugin failed to load")
	content := fmt.Sprintf("%s: %s", g.CoreAPI.Translate("error", "A plugin failed to load and may be unavailable until you reinstall or update it"), pkg)

	if err := g.CoreAPI.Notification().AddNotification(context.Background(), sdkapi.AddNotificationParams{
		Subject: subject,
		Content: content,
		Type:    sdkapi.NotificationTypeError,
	}); err != nil {
		if logErr := g.CoreAPI.Logger().Error(fmt.Sprintf("failed to notify admin of plugin load failure for %q: %v", pkg, err)); logErr != nil {
		}
	}
}

// notifyPluginBackupFailure records that recovery from backup also failed after a
// plugin failed to load — the plugin could not be restored to a working state and
// is now unavailable. The detailed cause is always logged; in production an admin
// notification is raised so the operator knows manual recovery (reinstall) is
// required. The user-facing message names only the package, never the underlying
// error or filesystem path, per the error-message hygiene rules.
func notifyPluginBackupFailure(g *api.CoreGlobals, pkg string, cause error) {
	if err := g.CoreAPI.Logger().Error(fmt.Sprintf("plugin %q failed to restore from backup: %v", pkg, cause)); err != nil {
	}

	// As in notifyPluginLoadFailure: only dev stays silent; every deployed device
	// notifies the operator that the plugin needs a reinstall.
	if env.IsDevEnv() {
		return
	}

	subject := g.CoreAPI.Translate("error", "Plugin recovery failed")
	content := fmt.Sprintf("%s: %s", g.CoreAPI.Translate("error", "A plugin failed to load and could not be restored from backup. Please reinstall it"), pkg)

	if err := g.CoreAPI.Notification().AddNotification(context.Background(), sdkapi.AddNotificationParams{
		Subject: subject,
		Content: content,
		Type:    sdkapi.NotificationTypeError,
	}); err != nil {
		if logErr := g.CoreAPI.Logger().Error(fmt.Sprintf("failed to notify admin of plugin backup recovery failure for %q: %v", pkg, err)); logErr != nil {
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

	// Load only (no Init) to match the main boot loop: a recovered plugin goes
	// through the same deferred-Init path (InitLoadedPlugins / provisioning).
	p := api.NewPluginApi(pkgInstallDir, info, g.GlobalAssets, g.PluginMgr, g.TrafficMgr, g.WifiMgr)
	if err := g.PluginMgr.LoadPlugin(p); err != nil {
		return err
	}

	plugins.RemoveBackup(pkg)

	return nil
}
