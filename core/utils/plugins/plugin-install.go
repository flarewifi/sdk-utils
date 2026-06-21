package plugins

import (
	"bytes"
	"core/utils/env"
	"core/utils/migrate"
	cmd "core/utils/shell"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func InstallSrcDef(db *sql.DB, def sdkutils.PluginSrcDef, opts InstallOpts) (info sdkutils.PluginInfo, err error) {
	switch def.Src {
	case sdkutils.PluginSrcGit:
		info, err = InstallFromGitSrc(db, def, opts)

	case sdkutils.PluginSrcLocal, sdkutils.PluginSrcSystem:
		info, err = InstallFromLocalPath(db, def, opts)

	case sdkutils.PluginSrcStore:
		// Store installs require a transient download URL that is not part of
		// the persisted PluginSrcDef. Callers must go through
		// PluginsMgr.InstallPlugin, which resolves the URL before installing.
		return sdkutils.PluginInfo{}, errors.New("store installs must be invoked via PluginsMgr.InstallPlugin")
	default:
		return sdkutils.PluginInfo{}, errors.New("Invalid plugin source: " + def.Src)
	}

	return info, err
}

func InstallFromLocalPath(db *sql.DB, def sdkutils.PluginSrcDef, opts InstallOpts) (info sdkutils.PluginInfo, err error) {
	sdkutils.PrettyPrint(def)

	info, err = sdkutils.GetPluginInfoFromPath(def.LocalPath)
	if err != nil {
		return
	}

	opts.RemoveSrc = false

	err = InstallPlugin(def.LocalPath, db, opts)
	if err != nil {
		return
	}

	return
}

// InstallFromPrebuilt installs a store plugin from a server-built, install-ready
// tarball (compiled plugin.so plus generated templates/queries/assets already
// included). The device only downloads and extracts — no on-device toolchain,
// no scratch disk. The URL is transient (resolved per-machine via
// RequestPluginBuild) and is not recorded on the persisted PluginSrcDef.
func InstallFromPrebuilt(sqldb *sql.DB, tarballURL string, opts InstallOpts) (sdkutils.PluginInfo, error) {
	if tarballURL == "" {
		return sdkutils.PluginInfo{}, errors.New("InstallFromPrebuilt: tarballURL is required")
	}

	workPath := filepath.Join(sdkutils.PathTmpDir, "plugins", "prebuilt", sdkutils.RandomStr(16))
	if err := sdkutils.FsEmptyDir(workPath); err != nil {
		return sdkutils.PluginInfo{}, err
	}
	defer os.RemoveAll(workPath)

	tarFile := filepath.Join(workPath, "package.tar.gz")
	if err := sdkutils.Download(tarballURL, tarFile); err != nil {
		return sdkutils.PluginInfo{}, fmt.Errorf("InstallFromPrebuilt: download: %w", err)
	}

	srcPath := filepath.Join(workPath, "src")
	if err := sdkutils.Untar(tarFile, srcPath); err != nil {
		return sdkutils.PluginInfo{}, fmt.Errorf("InstallFromPrebuilt: extract: %w", err)
	}

	info, err := sdkutils.GetPluginInfoFromPath(srcPath)
	if err != nil {
		return sdkutils.PluginInfo{}, fmt.Errorf("InstallFromPrebuilt: invalid package tree: %w", err)
	}

	if err := InstallPrebuilt(srcPath, sqldb, opts); err != nil {
		return sdkutils.PluginInfo{}, err
	}

	return info, nil
}

func InstallFromGitSrc(sqldb *sql.DB, def sdkutils.PluginSrcDef, opts InstallOpts) (sdkutils.PluginInfo, error) {
	clonePath := filepath.Join(sdkutils.PathTmpDir, "plugins", "cloned", sdkutils.RandomStr(16))

	repo := sdkutils.GitRepoSource{URL: def.GitURL, Ref: def.GitRef}

	if err := sdkutils.GitClone(repo, clonePath); err != nil {
		return sdkutils.PluginInfo{}, err
	}
	defer os.RemoveAll(clonePath)

	info, err := sdkutils.GetPluginInfoFromPath(clonePath)
	if err != nil {
		return sdkutils.PluginInfo{}, err
	}

	if err := InstallPlugin(sdkutils.StripRootPath(clonePath), sqldb, opts); err != nil {
		return sdkutils.PluginInfo{}, err
	}

	return info, nil
}

type InstallOpts struct {
	Def          sdkutils.PluginSrcDef
	RemoveSrc    bool
	ForceInstall bool
	// AsMetaMember, when non-empty, marks this install as a member pulled in by
	// the named meta plugin. The plugin's metadata records the meta as an owner
	// instead of flagging the plugin as a standalone (user-initiated) install.
	AsMetaMember string
	// PinnedDeps is the per-core-version dependency lock to build the plugin .so
	// against, so an on-device install is ABI-compatible with the running core and
	// the other installed plugins. Nil = build unpinned (the lock was unavailable or
	// empty). Callers in internal/api populate it via plugindeps.Fetch.
	PinnedDeps []LockedGoModule
}

func InstallPlugin(pluginSrc string, sqldb *sql.DB, opts InstallOpts) error {
	sdkutils.PrettyPrint(opts)

	parentpath := filepath.Join(sdkutils.PathTmpDir, "b", sdkutils.RandomStr(16))
	buildpath := filepath.Join(parentpath, "0")
	if err := sdkutils.FsEmptyDir(buildpath); err != nil {
		return err
	}
	defer os.RemoveAll(parentpath)

	if err := PatchPluginDeps(pluginSrc, opts.PinnedDeps); err != nil {
		return err
	}

	if err := BuildTemplates(pluginSrc); err != nil {
		return err
	}

	if err := RunMigrations(sqldb, pluginSrc); err != nil {
		return err
	}

	if err := BuildQueries(pluginSrc); err != nil {
		return err
	}

	if err := BuildAssets(pluginSrc); err != nil {
		return err
	}

	if err := BuildPluginSo(pluginSrc, buildpath, BuildOpts{PinnedDeps: opts.PinnedDeps}); err != nil {
		return err
	}

	info, err := sdkutils.GetPluginInfoFromPath(pluginSrc)
	if err != nil {
		return err
	}

	// Remove plugin.so if not core system to save space
	defer func() {
		coreInfo, err := sdkutils.GetPluginInfoFromPath(sdkutils.PathCoreDir)
		if err != nil {
			return
		}
		if info.Package != coreInfo.Package {
			os.RemoveAll(filepath.Join(pluginSrc, "plugin.so"))
		}
	}()

	installPath := GetInstallPath(info.Package)
	if !opts.ForceInstall {
		if err := ValidateInstallPath(installPath); err == nil {
			installPath = GetPendingUpdatePath(info.Package)
		}
	}

	if opts.ForceInstall {
		if err := os.RemoveAll(installPath); err != nil {
			return err
		}
		if err := os.RemoveAll(GetPendingUpdatePath(info.Package)); err != nil {
			return err
		}
	}

	if !opts.ForceInstall {
		// keep existing behavior: install to pending updates when already installed
		// and force install is not requested.
		installPath = GetPendingUpdatePath(info.Package)
	}

	if err := ProvisionSystemPkgs(info); err != nil {
		return err
	}

	if err := RunInstallScript(pluginSrc, info, info.PreInstall, "preinstall"); err != nil {
		return err
	}

	// Save the source path for local/system/zip installs. Git and store
	// installs intentionally do NOT persist LocalPath — pluginSrc is a
	// transient clone/mount path that is removed immediately after install.
	if opts.Def.LocalPath == "" &&
		opts.Def.Src != sdkutils.PluginSrcGit &&
		opts.Def.Src != sdkutils.PluginSrcStore {
		opts.Def.LocalPath = pluginSrc
	}

	// Preserve "store" registration: once a plugin's metadata is recorded as
	// Src=store (i.e., published to the marketplace), subsequent devel/local
	// installs must NOT overwrite that source back to "local". The publish
	// flag is a developer-facing decision that should survive boot-time
	// reinstalls of the devel source.
	shouldWriteMetadata := true
	if opts.Def.Src == sdkutils.PluginSrcLocal {
		if existing, readErr := ReadMetadata(info.Package); readErr == nil && existing.Def.Src == sdkutils.PluginSrcStore {
			shouldWriteMetadata = false
		}
	}
	if shouldWriteMetadata {
		if opts.AsMetaMember != "" {
			if err := WriteMetadataAsMember(opts.Def, info.Package); err != nil {
				return err
			}
		} else if err := WriteMetadata(opts.Def, info.Package); err != nil {
			return err
		}
	}

	if err := sdkutils.CopyPluginFiles(pluginSrc, installPath); err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Join(pluginSrc, "resources/assets/dist")) // Clean up dist folder

	if err := RunInstallScript(pluginSrc, info, info.PostInstall, "postinstall"); err != nil {
		return err
	}

	if opts.RemoveSrc {
		if err := os.RemoveAll(pluginSrc); err != nil {
			return err
		}
	}

	return nil
}

// InstallPrebuilt runs the non-build install steps for a server-built plugin
// tree: migrations, system packages, metadata, and the copy into the install
// path. The tree must already carry the compiled plugin.so and all generated
// artifacts — nothing is compiled or generated on the device.
func InstallPrebuilt(pluginSrc string, sqldb *sql.DB, opts InstallOpts) error {
	if err := RunMigrations(sqldb, pluginSrc); err != nil {
		return err
	}

	info, err := sdkutils.GetPluginInfoFromPath(pluginSrc)
	if err != nil {
		return err
	}

	installPath := GetInstallPath(info.Package)
	if opts.ForceInstall {
		if err := os.RemoveAll(installPath); err != nil {
			return err
		}
		if err := os.RemoveAll(GetPendingUpdatePath(info.Package)); err != nil {
			return err
		}
	} else {
		// Same behavior as InstallPlugin: stage as a pending update when not
		// force-installing.
		installPath = GetPendingUpdatePath(info.Package)
	}

	if err := ProvisionSystemPkgs(info); err != nil {
		return err
	}

	if err := RunInstallScript(pluginSrc, info, info.PreInstall, "preinstall"); err != nil {
		return err
	}

	if opts.AsMetaMember != "" {
		if err := WriteMetadataAsMember(opts.Def, info.Package); err != nil {
			return err
		}
	} else if err := WriteMetadata(opts.Def, info.Package); err != nil {
		return err
	}

	if err := sdkutils.CopyPluginFiles(pluginSrc, installPath); err != nil {
		return err
	}

	if err := RunInstallScript(pluginSrc, info, info.PostInstall, "postinstall"); err != nil {
		return err
	}

	if opts.RemoveSrc {
		if err := os.RemoveAll(pluginSrc); err != nil {
			return err
		}
	}

	return nil
}

// RunInstallScript executes an optional plugin lifecycle script
// (preinstall/postinstall) declared in plugin.json. scriptRel is the path
// relative to pluginDir; an empty value is a no-op. The script runs with
// pluginDir as its working directory so it can reference bundled files by
// relative path, and inherits the server's stdout/stderr for logging.
//
// On success it records a version-pinned marker (see ScriptMarkerPath) so the
// same script is not run again for this plugin version. os_image builds bake
// plugins in without ever calling this, so the marker is absent on a device's
// first boot and the boot path (RunInstallScripts) runs the script then.
//
// The current build environment is exposed to the script as GO_ENV
// ("development" | "sandbox" | "staging" | "production"), so a script can guard
// on it (e.g. `[ "$GO_ENV" = production ] || exit 0`) to skip device-only setup
// in development. The script also inherits the server's stdout/stderr.
func RunInstallScript(pluginDir string, info sdkutils.PluginInfo, scriptRel, phase string) error {
	if scriptRel == "" {
		return nil
	}

	// Resolve absolute paths: the working directory is set to the plugin dir, so
	// passing the script as a path already joined with (a possibly relative)
	// pluginDir would resolve it twice. An absolute script path + absolute Dir
	// avoids that, regardless of whether pluginDir is relative.
	scriptPath := filepath.Join(pluginDir, scriptRel)
	if !sdkutils.FsExists(scriptPath) {
		return fmt.Errorf("%s script not found: %s", phase, scriptRel)
	}
	absScript, err := filepath.Abs(scriptPath)
	if err != nil {
		absScript = scriptPath
	}
	absDir, err := filepath.Abs(pluginDir)
	if err != nil {
		absDir = pluginDir
	}

	fmt.Printf("[plugin-install] running %s script: %s\n", phase, scriptRel)
	if err := cmd.Exec("sh \""+absScript+"\"", &cmd.ExecOpts{
		Dir:    absDir,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    append(os.Environ(), "GO_ENV="+env.GoEnvString()),
	}); err != nil {
		return fmt.Errorf("%s script failed: %w", phase, err)
	}

	// Pin the marker to the plugin version that just ran. A write failure is not
	// fatal to the install — at worst the script runs once more on the next boot,
	// which the scripts are required to tolerate (idempotent / self-guarding).
	if err := WriteScriptMarker(info.Package, phase, info.Version); err != nil {
		fmt.Printf("[plugin-install] warning: could not record %s marker for %s: %v\n", phase, info.Package, err)
	}

	return nil
}

// ScriptMarkerPath returns the persistent marker file that records the plugin
// version for which a given install-script phase (preinstall/postinstall) last
// completed. It lives under data/storage (not the plugin install dir) so it
// survives a plugin update that replaces the install dir; the recorded version
// is what makes the run both once-per-version and re-run on upgrade.
func ScriptMarkerPath(pkg, phase string) string {
	return filepath.Join(sdkutils.PathPluginStorageDir, pkg, "install-scripts", phase)
}

// ReadScriptMarker returns the plugin version recorded for a completed script
// phase, or "" when the script has never run for this plugin (marker
// absent/empty) — e.g. on the first boot of an os_image that baked the plugin in.
func ReadScriptMarker(pkg, phase string) string {
	b, err := os.ReadFile(ScriptMarkerPath(pkg, phase))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// WriteScriptMarker records that the given phase's script completed for version.
func WriteScriptMarker(pkg, phase, version string) error {
	markerPath := ScriptMarkerPath(pkg, phase)
	if err := os.MkdirAll(filepath.Dir(markerPath), sdkutils.PermDir); err != nil {
		return err
	}
	return os.WriteFile(markerPath, []byte(version), sdkutils.PermFile)
}

// ProvisionSystemPkgs installs a plugin's declared system_packages exactly once
// per plugin version, recording a version-pinned "syspkgs" marker (see
// ScriptMarkerPath) so the opkg work is not repeated on every boot or every
// internet-up. It requires connectivity — InstallSystemPkgs runs `opkg update`,
// which hits the package feed — so callers must invoke it only when the device is
// online: at runtime install time (the dashboard download already proves egress),
// or from the online monitor's internet-up provisioning pass. A version mismatch
// (fresh install or upgrade) re-runs it.
func ProvisionSystemPkgs(info sdkutils.PluginInfo) error {
	if len(info.SystemPackages) == 0 {
		return nil
	}
	if ReadScriptMarker(info.Package, "syspkgs") == info.Version {
		return nil
	}
	if err := InstallSystemPkgs(info.SystemPackages); err != nil {
		return err
	}
	if err := WriteScriptMarker(info.Package, "syspkgs", info.Version); err != nil {
		fmt.Printf("[plugin-install] warning: could not record syspkgs marker for %s: %v\n", info.Package, err)
	}
	return nil
}

// systemPkgsInstallAttempts is how many times InstallSystemPkgs retries the opkg
// work before giving up. Right after boot / an internet-up transition the link is
// often not yet stable, so `opkg update` (which hits the package feed) can fail
// transiently; retrying with backoff lets the install succeed once connectivity
// settles instead of waiting for the next internet-up pass.
const systemPkgsInstallAttempts = 5

func InstallSystemPkgs(packages []string) error {
	if len(packages) == 0 {
		return nil
	}

	// Retry the whole opkg sequence: 2s, 4s, 6s, 8s backoff between attempts
	// (see sdkutils.Retry) absorbs the network still stabilizing after boot.
	_, err := sdkutils.Retry(func() (struct{}, error) {
		return struct{}{}, installSystemPkgsOnce(packages)
	}, systemPkgsInstallAttempts)
	return err
}

// installSystemPkgsOnce performs a single attempt of the opkg update + install
// sequence. It is the unit retried by InstallSystemPkgs.
func installSystemPkgsOnce(packages []string) error {
	if err := cmd.Exec("opkg update", &cmd.ExecOpts{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}); err != nil {
		return err
	}

	toBeInstalled := []string{}
	for _, pkg := range packages {
		installed, err := IsSystemPackageInstalled(pkg)
		if err != nil {
			return err
		}
		if !installed {
			toBeInstalled = append(toBeInstalled, pkg)
		}
	}

	if len(toBeInstalled) == 0 {
		return nil
	}

	if err := cmd.Exec("opkg install "+strings.Join(toBeInstalled, " "), &cmd.ExecOpts{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}); err != nil {
		return err
	}

	return nil
}

// IsSystemPackageInstalled checks if a package is installed on OpenWrt. It runs
// through the shell wrapper (not a raw exec) so the dev build can stub opkg out.
func IsSystemPackageInstalled(opkgPackage string) (bool, error) {
	var output bytes.Buffer
	if err := cmd.ExecOutput("opkg list-installed", &output); err != nil {
		return false, fmt.Errorf("failed to execute opkg: %v, output: %s", err, output.String())
	}

	// opkg list-installed lines look like "pkgname - version"; match the package
	// name exactly so e.g. "python3" doesn't satisfy "python3-light".
	for _, line := range strings.Split(output.String(), "\n") {
		name := strings.TrimSpace(line)
		if i := strings.IndexByte(name, ' '); i >= 0 {
			name = name[:i]
		}
		if name == opkgPackage {
			return true, nil
		}
	}
	return false, nil
}

func RunMigrations(sqldb *sql.DB, pluginDir string) (err error) {
	_, err = sdkutils.GetPluginInfoFromPath(pluginDir)
	if err != nil {
		return
	}

	err = migrate.MigrateUp(sqldb, pluginDir)
	if err != nil {
		return err
	}

	return nil
}

func MarkToRemove(pkg string) error {
	installPath := GetInstallPath(pkg)
	if !sdkutils.FsExists(installPath) {
		return errors.New("Plugin not installed: " + pkg)
	}
	uninstallFile := filepath.Join(installPath, "uninstall")
	return os.WriteFile(uninstallFile, []byte(""), sdkutils.PermFile)
}

func IsToBeRemoved(pkg string) bool {
	installPath := GetInstallPath(pkg)
	uninstallFile := filepath.Join(installPath, "uninstall")
	return sdkutils.FsExists(uninstallFile)
}

func UninstallPlugin(pkg string, sqldb *sql.DB) error {
	meta, err := ReadMetadata(pkg)
	metaFound := err == nil

	if metaFound {
		if err := RemoveMetadata(pkg); err != nil {
			return err
		}
	}

	installPath := GetInstallPath(pkg)

	// Only run down-migrations if the plugin has a migrations directory.
	migDir := filepath.Join(installPath, "resources/migrations")
	if sdkutils.FsExists(migDir) {
		if err := migrate.MigrateDown(installPath, sqldb); err != nil {
			return err
		}
	}

	// Only remove the source directory for non-local/system plugins (e.g. git, store, zip).
	// For local/system plugins the source is the plugin repo itself and must not be deleted.
	if metaFound && meta.Def.Src != sdkutils.PluginSrcLocal && meta.Def.Src != sdkutils.PluginSrcSystem {
		if meta.Def.LocalPath != "" {
			if err := os.RemoveAll(meta.Def.LocalPath); err != nil {
				return err
			}
		}
	}

	if err := os.RemoveAll(installPath); err != nil {
		return err
	}

	return nil
}
