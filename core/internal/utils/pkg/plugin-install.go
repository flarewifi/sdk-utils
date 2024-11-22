package pkg

import (
	"core/internal/utils/download"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"

	sdkextract "github.com/flarehotspot/go-utils/extract"

	"core/internal/utils/encdisk"
	"core/internal/utils/git"
	sdkplugin "sdk/api/plugin"

	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
	sdkstr "github.com/flarehotspot/go-utils/strings"
)

type PluginMetadata struct {
	Def PluginSrcDef
}

type PluginFile struct {
	File     string
	Optional bool
}

var PLuginFiles = []PluginFile{
	{File: "LICENSE.txt", Optional: false},
	{File: "plugin.json", Optional: false},
	{File: "plugin.so", Optional: false},
	{File: "metadata.json", Optional: true},
	{File: "resources/assets/dist", Optional: true},
	{File: "resources/migrations", Optional: true},
	{File: "resources/translations", Optional: true},
	{File: "go.mod", Optional: false},
}

func InstallSrcDef(w io.Writer, def PluginSrcDef) (info sdkplugin.PluginInfo, err error) {
	switch def.Src {
	case PluginSrcGit:
		info, err = InstallFromGitSrc(w, def)
	case PluginSrcLocal, PluginSrcSystem:
		info, err = InstallFromLocalPath(w, def)
	case PluginSrcZip:
		info, err = InstallFromZipFile(w, def)
	case PluginSrcStore:
		info, err = InstallFromPluginStore(w, def)
	default:
		return sdkplugin.PluginInfo{}, errors.New("Invalid plugin source: " + def.Src)
	}

	return info, err
}

func InstallFromLocalPath(w io.Writer, def PluginSrcDef) (sdkplugin.PluginInfo, error) {
	w.Write([]byte("Installing plugin from local path: " + def.LocalPath))

	info, err := GetSrcInfo(def.LocalPath)
	if err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	err = InstallPlugin(def.LocalPath, InstallOpts{Def: def, RemoveSrc: false})
	if err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	return info, nil
}

func InstallFromZipFile(w io.Writer, def PluginSrcDef) (sdkplugin.PluginInfo, error) {
	w.Write([]byte("Installing zipped plugin from local path: " + def.LocalPath))

	// prepare path
	randomPath := RandomPluginPath()
	workPath := filepath.Join(randomPath, "workpath")

	// extract compressed plugin release
	sdkextract.Extract(def.LocalZipFile, workPath)

	os.RemoveAll(filepath.Dir(def.LocalZipFile))

	// gets the plugin release source path
	newWorkPath, err := FindPluginSrc(workPath)
	if err != nil {
		err = errors.New("Unable to find plugin source in: " + workPath)
		log.Println("Error: ", err)
		return sdkplugin.PluginInfo{}, err
	}

	// read the plugin.json
	info, err := GetSrcInfo(newWorkPath)
	if err != nil {
		log.Println("Error getting plugin info: ", err)
		return sdkplugin.PluginInfo{}, err
	}

	def.LocalPath = filepath.Join(GetInstallPath(info.Package))

	if err := InstallPlugin(newWorkPath, InstallOpts{Def: def, RemoveSrc: false}); err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	return info, nil
}

func InstallFromPluginStore(w io.Writer, def PluginSrcDef) (sdkplugin.PluginInfo, error) {
	w.Write([]byte("Installing plugin from store: " + def.StorePackage))

	// prepare path
	randomPath := RandomPluginPath()
	diskfile := filepath.Join(randomPath, "disk")
	mountpath := filepath.Join(randomPath, "mount")
	clonePath := filepath.Join(mountpath, "clone", "0") // need extra sub dir
	workPath := filepath.Join(mountpath, "clone", "1")  // need extra sub dir

	// prepare encrypted virtual disk path
	dev := sdkstr.Rand(8)
	mnt := encdisk.NewEncrypedDisk(diskfile, mountpath, dev)
	if err := mnt.Mount(); err != nil {
		log.Println("Error mounting disk: ", err)
		return sdkplugin.PluginInfo{}, err
	}
	defer mnt.Unmount()

	// download plugin release zip file
	log.Println("downloading plugin release: ", def.StoreZipUrl)
	downloader := download.NewDownloader(def.StoreZipUrl, clonePath)
	if err := downloader.Download(); err != nil {
		log.Println("Error: ", err)
		return sdkplugin.PluginInfo{}, err
	}

	// extract compressed plugin release
	sdkextract.Extract(clonePath, workPath)

	// clear StoreZipUrl def
	def.StoreZipUrl = ""

	newWorkPath, err := FindPluginSrc(workPath)
	if err != nil {
		err = errors.New("Unable to find plugin source in: " + workPath)
		log.Println("Error: ", err)
		return sdkplugin.PluginInfo{}, err
	}
	info, err := GetSrcInfo(newWorkPath)
	if err != nil {
		log.Println("Error getting plugin info: ", err)
		return sdkplugin.PluginInfo{}, err
	}

	if err := InstallPlugin(newWorkPath, InstallOpts{Def: def, RemoveSrc: false}); err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	return info, nil
}

func InstallFromGitSrc(w io.Writer, def PluginSrcDef) (sdkplugin.PluginInfo, error) {
	log.Println("Installing plugin from git source: " + def.String())
	randomPath := RandomPluginPath()
	diskfile := filepath.Join(randomPath, "disk")
	mountpath := filepath.Join(randomPath, "mount")
	clonePath := filepath.Join(mountpath, "clone", "0") // need extra sub dir

	dev := sdkstr.Rand(8)
	mnt := encdisk.NewEncrypedDisk(diskfile, mountpath, dev)
	if err := mnt.Mount(); err != nil {
		log.Println("Error mounting disk: ", err)
		return sdkplugin.PluginInfo{}, err
	}

	defer mnt.Unmount()

	repo := git.RepoSource{URL: def.GitURL, Ref: def.GitRef}

	log.Println("Cloning plugin from git: " + def.GitURL)
	if err := git.Clone(w, repo, clonePath); err != nil {
		log.Println("Error cloning: ", err)
		return sdkplugin.PluginInfo{}, err
	}

	info, err := GetSrcInfo(clonePath)
	if err != nil {
		log.Println("Error getting plugin info: ", err)
		return sdkplugin.PluginInfo{}, err
	}

	if err := InstallPlugin(clonePath, InstallOpts{Def: def, RemoveSrc: false}); err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	return info, nil
}

func InstallPlugin(src string, opts InstallOpts) error {
	log.Println("Installing plugin: ", src)

	var buildpath string

	if opts.Encrypt {
		dev := sdkstr.Rand(8)
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
		parentpath := filepath.Join(sdkpaths.TmpDir, "b", sdkstr.Rand(16))
		buildpath = filepath.Join(parentpath, "0")
		if err := sdkfs.EmptyDir(buildpath); err != nil {
			return err
		}
		defer os.RemoveAll(parentpath)
	}

	if err := BuildPluginSo(src, buildpath); err != nil {
		log.Println("Error building plugin: ", err)
		return err
	}

	info, err := GetSrcInfo(src)
	if err != nil {
		return err
	}

	installPath := GetInstallPath(info.Package)
	if err := ValidateInstallPath(installPath); err == nil {
		installPath = GetPendingUpdatePath(info.Package)
	}

	if err := WriteMetadata(opts.Def, installPath); err != nil {
		return err
	}

	log.Println("Copying plugin files to: ", installPath)
	for _, f := range PLuginFiles {
		err := sdkfs.Copy(filepath.Join(src, f.File), filepath.Join(installPath, f.File))
		if err != nil && !f.Optional {
			return err
		}
	}

	if opts.RemoveSrc {
		if err := os.RemoveAll(src); err != nil {
			return err
		}
	}

	log.Println("Plugin installed")

	return nil
}

func MarkToRemove(pkg string) error {
	installPath := GetInstallPath(pkg)
	if !sdkfs.Exists(installPath) {
		return errors.New("Plugin not installed: " + pkg)
	}
	uninstallFile := filepath.Join(installPath, "uninstall")
	return os.WriteFile(uninstallFile, []byte(""), sdkfs.PermFile)
}

func IsToBeRemoved(pkg string) bool {
	uninstallFile := filepath.Join(GetInstallPath(pkg), "uninstall")
	return sdkfs.Exists(uninstallFile)
}

func RemovePlugin(pack string) error {
	metadata, err := ReadMetadata(pack)
	if err != nil {
		return err
	}
	def := metadata.Def
	if def.Src == PluginSrcLocal || def.Src == PluginSrcSystem {
		return os.RemoveAll(def.LocalPath)
	}
	if err := os.RemoveAll(GetInstallPath(pack)); err != nil {
		return err
	}
	return nil
}
