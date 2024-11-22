package pkg

import (
	"core/env"
	"core/internal/config"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-json"

	sdkfs "github.com/flarehotspot/go-utils/fs"
	paths "github.com/flarehotspot/go-utils/paths"
)

var (
	ErrNotInstalled = errors.New("Plugin is not installed")
)

const (
	PluginSrcGit          string = "git"
	PluginSrcStore        string = "store"
	PluginSrcSystem       string = "system"
	PluginSrcLocal        string = "local"
	PluginSrcZip          string = "zip"
	pluginsConfigJsonFile string = "plugins.json"
)

type PluginSrc string

type PluginInstallData struct {
	Def         PluginSrcDef
	InstallPath string
}

type PluginDefList []PluginSrcDef

func PluginsUserList() PluginDefList {
	configFile := filepath.Join(paths.ConfigDir, pluginsConfigJsonFile)
	bytes, err := os.ReadFile(configFile)
	if err != nil {
		return PluginDefList{}
	}

	var userJson PluginDefList

	err = json.Unmarshal(bytes, &userJson)
	if err != nil {
		return PluginDefList{}
	}

	return userJson
}

func IsDefInList(defs PluginDefList, def PluginSrcDef) bool {
	for _, i := range defs {
		if i.Equal(def) {
			return true
		}
	}
	return false
}

func AllPluginDef() PluginDefList {
	list := InsalledPluginsDef()
	localPlugins := LocalPlugins()
	for _, loc := range localPlugins {
		if !IsDefInList(list, loc) {
			list = append(list, loc)
		}
	}
	return list
}

func LocalPlugins() PluginDefList {
	var list PluginDefList
	paths := LocalPluginPaths()
	for _, p := range paths {
		list = append(list, PluginSrcDef{Src: PluginSrcLocal, LocalPath: p})
	}
	log.Println("local plugins list: ", list)
	return list
}

func InsalledPluginsDef() PluginDefList {
	var list PluginDefList
	paths := InstalledDirList()
	for _, p := range paths {
		info, err := GetSrcInfo(p)
		if err != nil {
			log.Println("Error reading plugin info: ", err)
			continue
		}
		metadata, err := ReadMetadata(info.Package)
		list = append(list, metadata.Def)
	}
	return list
}

// LocalPluginPaths returns a list of plugin absolute source paths
func LocalPluginPaths() []string {
	searchPaths := []string{"plugins/system", "plugins/local"}
	pluginPaths := []string{}

	for _, sp := range searchPaths {
		if sdkfs.Exists(sp) {
			var dirs []string
			if err := sdkfs.LsDirs(sp, &dirs, false); err != nil {
				continue
			}

			for _, dir := range dirs {
				pluginJson := filepath.Join(dir, "plugin.json")
				modFile := filepath.Join(dir, "go.mod")

				if sdkfs.Exists(pluginJson) && sdkfs.Exists(modFile) {
					pluginPaths = append(pluginPaths, dir)
				}
			}
		}
	}

	return pluginPaths
}

// InstalledDirList returns the list of installed plugins in the plugins directory. The path of each plugin is an aboslute path.
func InstalledDirList() []string {
	var pluginList []string

	installedPluginsPath := filepath.Join(paths.PluginsDir, "installed")

	// check if plugins/installed directory exists before traversing
	if !(sdkfs.Exists(installedPluginsPath)) {
		return pluginList
	}

	// this lists all directories inside paths.PluginsDir/installed
	if err := sdkfs.LsDirs(installedPluginsPath, &pluginList, false); err != nil {
		panic(err)
	}

	return pluginList
}

func WriteMetadata(def PluginSrcDef, installPath string) error {
	metapath := filepath.Join(installPath, "metadata.json")
	metadata := PluginMetadata{
		Def: def,
	}

	if err := sdkfs.EnsureDir(installPath); err != nil {
		return err
	}

	return sdkfs.WriteJson(metapath, metadata)
}

func IsPackageInstalled(pkg string) bool {
	installPath := GetInstallPath(pkg)
	err := ValidateInstallPath(installPath)
	return err == nil
}

func IsSrcDefInstalled(def PluginSrcDef) bool {
	installPath, ok := FindDefInstallPath(def)
	if !ok {
		return false
	}
	err := ValidateInstallPath(installPath)
	return err == nil
}

func InstalledPluginsList() []PluginInstallData {
	marks := []PluginInstallData{}
	list := InstalledDirList()
	for _, p := range list {
		info, err := GetSrcInfo(p)
		if err != nil {
			log.Println("Error reading plugin info: ", err)
			continue
		}
		metadata, err := ReadMetadata(info.Package)
		if err != nil {
			log.Println("Error reading plugin metadata: ", err)
			continue
		}

		marks = append(marks, PluginInstallData{
			Def:         metadata.Def,
			InstallPath: p,
		})
	}
	return marks
}

func NeedsRecompile(def PluginSrcDef) bool {
	if env.GO_ENV == env.ENV_DEV && (def.Src == PluginSrcLocal || def.Src == PluginSrcSystem) {
		return true
	}

	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		log.Println("Error reading plugins config: ", err)
		return true
	}

	path, ok := FindDefInstallPath(def)
	if !ok {
		log.Println("Plugin is not installed: ", def.LocalPath)
		return true
	}

	info, err := GetSrcInfo(path)
	if err != nil {
		return true
	}

	for _, pkg := range cfg.Recompile {
		if info.Package == pkg {
			return true
		}
	}

	return false
}

func HasPendingUpdate(pkg string) bool {
	updatepath := GetPendingUpdatePath(pkg)
	return ValidateInstallPath(updatepath) == nil
}

func MovePendingUpdate(pkg string) error {
	updatePath := GetPendingUpdatePath(pkg)
	if err := CreateBackup(pkg); err != nil {
		return err
	}
	if err := sdkfs.Copy(updatePath, GetInstallPath(pkg)); err != nil {
		if err := RestoreBackup(pkg); err != nil {
			return err
		}
		return err
	}
	if err := os.RemoveAll(updatePath); err != nil {
		return err
	}
	if HasBackup(pkg) {
		if err := RemoveBackup(pkg); err != nil {
			return err
		}
	}
	return nil
}

func CreateBackup(pkg string) error {
	installPath := GetInstallPath(pkg)
	backupPath := GetBackupPath(pkg)
	if sdkfs.Exists(backupPath) {
		if err := os.RemoveAll(backupPath); err != nil {
			return err
		}
	}
	return sdkfs.Copy(installPath, backupPath)
}

func HasBackup(pkg string) bool {
	backup := GetBackupPath(pkg)
	err := ValidateInstallPath(backup)
	return err == nil
}

func RestoreBackup(pkg string) error {
	backupPath := GetBackupPath(pkg)
	if err := sdkfs.Copy(backupPath, GetInstallPath(pkg)); err != nil {
		return err
	}
	if err := os.RemoveAll(backupPath); err != nil {
		return err
	}
	return nil
}

func RemoveBackup(pkg string) error {
	return os.RemoveAll(GetBackupPath(pkg))
}

func RemovePendingUpdate(pkg string) error {
	return os.RemoveAll(GetPendingUpdatePath(pkg))
}

func ReadMetadata(pkg string) (PluginMetadata, error) {
	var metadata PluginMetadata
	installPath := GetInstallPath(pkg)
	err := sdkfs.ReadJson(filepath.Join(installPath, "metadata.json"), &metadata)
	return metadata, err
}

func ValidateSrcPath(src string) error {
	requiredFiles := []string{"plugin.json", "go.mod", "main.go"}

	for _, f := range requiredFiles {
		if !sdkfs.Exists(filepath.Join(src, f)) {
			return errors.New(f + " not found in source path")
		}
	}
	return nil
}

func ValidateInstallPath(src string) error {
	requiredFiles := []string{"plugin.json", "go.mod", "plugin.so", "metadata.json"}

	for _, f := range requiredFiles {
		if !sdkfs.Exists(filepath.Join(src, f)) {
			return errors.New(f + " not found in source path")
		}
	}
	return nil
}

func FindPluginSrc(dir string) (string, error) {
	files := []string{}
	err := sdkfs.LsFiles(dir, &files, true)
	if err != nil {
		return dir, err
	}

	for _, f := range files {
		if filepath.Base(f) == "plugin.json" {
			return filepath.Dir(f), nil
		}
	}

	return "", errors.New("Can't find plugin.json in " + paths.StripRoot(dir))
}

func GetAuthorNameFromGitUrl(p PluginInstallData) string {
	return strings.Split(strings.TrimPrefix(p.Def.GitURL, "https://github.com/"), "/")[0]
}

func GetRepoFromGitUrl(p PluginInstallData) string {
	return strings.Split(strings.TrimPrefix(p.Def.GitURL, fmt.Sprintf("https://github.com/%s/", GetAuthorNameFromGitUrl(p))), "/")[0]
}
