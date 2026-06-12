package plugins

import (
	"bytes"
	"core/utils/migrate"
	cmd "core/utils/shell"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
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

	if err := InstallSystemPkgs(info.SystemPackages); err != nil {
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

	if err := InstallSystemPkgs(info.SystemPackages); err != nil {
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

	if opts.RemoveSrc {
		if err := os.RemoveAll(pluginSrc); err != nil {
			return err
		}
	}

	return nil
}

func InstallSystemPkgs(packages []string) (err error) {
	if len(packages) == 0 {
		return nil
	}

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

	if err := cmd.Exec("opkg install "+strings.Join(toBeInstalled, " "), &cmd.ExecOpts{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}); err != nil {
		return err
	}

	return nil
}

// IsPackageInstalled checks if a package is installed on OpenWrt.
func IsSystemPackageInstalled(opkgPackage string) (bool, error) {
	// Execute the `opkg list-installed` command
	cmd := exec.Command("opkg", "list-installed")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("failed to execute opkg: %v, output: %s", err, output.String())
	}

	// Check if the package name exists in the output
	installedPackages := output.String()
	return strings.Contains(installedPackages, opkgPackage), nil
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
