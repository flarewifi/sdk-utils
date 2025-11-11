package plugins

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"tools/download"
	"tools/encdisk"
	"tools/migrate"
	cmd "tools/shell"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func InstallSrcDef(db *sql.DB, def sdkutils.PluginSrcDef, opts InstallOpts) (info sdkutils.PluginInfo, err error) {
	switch def.Src {
	case sdkutils.PluginSrcZip:
		info, err = InstallFromLocalPath(db, def, opts)

	case sdkutils.PluginSrcGit:
		info, err = InstallFromGitSrc(db, def, opts)

	case sdkutils.PluginSrcLocal, sdkutils.PluginSrcSystem:
		info, err = InstallFromLocalPath(db, def, opts)

	case sdkutils.PluginSrcStore:
		opts.RemoveSrc = true
		info, err = InstallFromPluginStore(db, def, opts)
	default:
		return sdkutils.PluginInfo{}, errors.New("Invalid plugin source: " + def.Src)
	}

	return info, err
}

func InstallFromLocalPath(db *sql.DB, def sdkutils.PluginSrcDef, opts InstallOpts) (info sdkutils.PluginInfo, err error) {
	fmt.Println("Installing plugin from local path: " + def.LocalPath)
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

func InstallFromPluginStore(sqldb *sql.DB, def sdkutils.PluginSrcDef, opts InstallOpts) (sdkutils.PluginInfo, error) {
	log.Println("Installing plugin from store: " + def.String())

	// prepare path
	randomPath := RandomPluginPath()
	diskfile := filepath.Join(randomPath, "disk")
	mountpath := filepath.Join(randomPath, "mount")
	clonePath := filepath.Join(mountpath, "clone", "0") // need extra sub dir
	workPath := filepath.Join(mountpath, "clone", "1")  // need extra sub dir

	// prepare encrypted virtual disk path
	dev := sdkutils.RandomStr(8)
	mnt := encdisk.NewEncrypedDisk(diskfile, mountpath, dev)
	if err := mnt.Mount(); err != nil {
		log.Println("Error mounting disk: ", err)
		return sdkutils.PluginInfo{}, err
	}
	defer mnt.Unmount()

	// download plugin release zip file
	log.Println("downloading plugin release: ", def.StoreZipURL)
	downloader := download.NewDownloader(def.StoreZipURL, clonePath)
	if err := downloader.Download(); err != nil {
		log.Println("Error: ", err)
		return sdkutils.PluginInfo{}, err
	}

	// extract compressed plugin release
	sdkutils.FsExtract(clonePath, workPath)

	// clear StoreZipURL def
	def.StoreZipURL = ""

	newWorkPath, err := sdkutils.FindPluginSrc(workPath)
	if err != nil {
		err = errors.New("Unable to find plugin source in: " + workPath)
		log.Println("Error: ", err)
		return sdkutils.PluginInfo{}, err
	}
	info, err := sdkutils.GetPluginInfoFromPath(newWorkPath)
	if err != nil {
		log.Println("Error getting plugin info: ", err)
		return sdkutils.PluginInfo{}, err
	}

	if err := InstallPlugin(newWorkPath, sqldb, opts); err != nil {
		return sdkutils.PluginInfo{}, err
	}

	return info, nil
}

func InstallFromGitSrc(sqldb *sql.DB, def sdkutils.PluginSrcDef, opts InstallOpts) (sdkutils.PluginInfo, error) {
	log.Println("Installing plugin from git source: " + def.String())
	clonePath := filepath.Join(sdkutils.PathTmpDir, "plugins", "cloned", sdkutils.RandomStr(16))

	repo := sdkutils.GitRepoSource{URL: def.GitURL, Ref: def.GitRef}

	log.Println("Cloning plugin from git: " + def.GitURL)
	if err := sdkutils.GitClone(repo, clonePath); err != nil {
		log.Println("Error cloning: ", err)
		return sdkutils.PluginInfo{}, err
	}
	defer os.RemoveAll(clonePath)

	info, err := sdkutils.GetPluginInfoFromPath(clonePath)
	if err != nil {
		log.Println("Error getting plugin info: ", err)
		return sdkutils.PluginInfo{}, err
	}

	cachePath := filepath.Join(sdkutils.PathPluginCacheDir, info.Package)
	if err := sdkutils.FsCopy(clonePath, cachePath); err != nil {
		return sdkutils.PluginInfo{}, err
	}

	if err := InstallPlugin(sdkutils.StripRootPath(cachePath), sqldb, opts); err != nil {
		return sdkutils.PluginInfo{}, err
	}

	return info, nil
}

type InstallOpts struct {
	Def          sdkutils.PluginSrcDef
	RemoveSrc    bool
	Encrypt      bool
	ForceInstall bool
}

func InstallPlugin(pluginSrc string, sqldb *sql.DB, opts InstallOpts) error {
	log.Println("Installing plugin: ", pluginSrc)
	sdkutils.PrettyPrint(opts)

	var buildpath string

	if opts.Encrypt {
		dev := sdkutils.RandomStr(8)
		parentPath := RandomPluginPath()
		diskfile := filepath.Join(parentPath, "disk")
		mountpath := filepath.Join(parentPath, "mount")
		buildpath = filepath.Join(mountpath, "build")
		mnt := encdisk.NewEncrypedDisk(diskfile, mountpath, dev)
		if err := mnt.Mount(); err != nil {
			log.Println("Error mounting: ", err)
			return err
		}

		defer mnt.Unmount()
	} else {
		parentpath := filepath.Join(sdkutils.PathTmpDir, "b", sdkutils.RandomStr(16))
		buildpath = filepath.Join(parentpath, "0")
		if err := sdkutils.FsEmptyDir(buildpath); err != nil {
			return err
		}
		defer os.RemoveAll(parentpath)
	}

	if err := PatchPluginDeps(pluginSrc); err != nil {
		return err
	}

	if err := BuildTemplates(pluginSrc); err != nil {
		log.Println("Error building plugin templates: ", err)
		return err
	}

	if err := RunMigrations(sqldb, pluginSrc); err != nil {
		log.Println("Error running migrations: ", err)
		return err
	}

	if err := BuildQueries(pluginSrc); err != nil {
		log.Println("Error building plugin sqlc: ", err)
		return err
	}

	if err := BuildAssets(pluginSrc); err != nil {
		return err
	}

	if err := BuildPluginSo(pluginSrc, buildpath); err != nil {
		log.Println("Error building plugin: ", err)
		return err
	}

	info, err := sdkutils.GetPluginInfoFromPath(pluginSrc)
	if err != nil {
		log.Println("Error building plugin: ", err)
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
	if err := ValidateInstallPath(installPath); err == nil {
		installPath = GetPendingUpdatePath(info.Package)
	}

	if err := InstallSystemPkgs(info.SystemPackages); err != nil {
		return err
	}

	// Save the source path
	if opts.Def.LocalPath == "" {
		opts.Def.LocalPath = pluginSrc
	}

	if err := WriteMetadata(opts.Def, info.Package); err != nil {
		log.Println("Error building plugin: ", err)
		return err
	}

	if opts.ForceInstall {
		if err := os.RemoveAll(installPath); err != nil {
			return err
		}
	}

	log.Println("Copying plugin files to: ", installPath)
	if err := sdkutils.CopyPluginFiles(pluginSrc, installPath); err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Join(pluginSrc, "resources/assets/dist")) // Clean up dist folder

	if opts.RemoveSrc {
		if err := os.RemoveAll(pluginSrc); err != nil {
			log.Println("Error building plugin: ", err)
			return err
		}
	}

	log.Println("Plugin installed")

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
	info, err := sdkutils.GetPluginInfoFromPath(pluginDir)
	if err != nil {
		return
	}

	name := info.Name

	err = migrate.MigrateUp(sqldb, pluginDir)
	if err != nil {
		log.Println("Error in plugin migration "+name, ":", err.Error())
		return err
	}

	log.Println("Done migrating plugin:", name)
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
	if err != nil {
		return err
	}

	if err := RemoveMetadata(pkg); err != nil {
		return err
	}

	installPath := GetInstallPath(pkg)
	if err := migrate.MigrateDown(installPath, sqldb); err != nil {
		return err
	}

	if meta.Def.Src == sdkutils.PluginSrcLocal || meta.Def.Src == sdkutils.PluginSrcSystem {
		if err := os.RemoveAll(meta.Def.LocalPath); err != nil {
			return err
		}
	}

	if err := os.RemoveAll(installPath); err != nil {
		return err
	}

	return nil
}
