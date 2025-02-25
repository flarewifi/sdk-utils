package plugins

import (
	"core/env"
	"core/internal/config"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	ErrNotInstalled = errors.New("Plugin is not installed")
)

func IsDefInList(defs []sdkutils.PluginSrcDef, def sdkutils.PluginSrcDef) bool {
	for _, i := range defs {
		if i.Equal(def) {
			return true
		}
	}
	return false
}

func AllPluginSrcDefs() []sdkutils.PluginSrcDef {
	list := InstalledPluginsDef()
	localPlugins := LocalPluginSrcDefs()
	systemPlugins := SystemPluginSrcDefs()
	alldefs := append(systemPlugins, localPlugins...)

	for _, loc := range alldefs {
		if !IsDefInList(list, loc) {
			list = append(list, loc)
		}
	}

	return list
}

func LocalPluginSrcDefs() []sdkutils.PluginSrcDef {
	list := []sdkutils.PluginSrcDef{}
	paths := SearchPluginDirs(filepath.Join("plugins", "local"))
	for _, p := range paths {
		list = append(list, sdkutils.PluginSrcDef{
			Src:       sdkutils.PluginSrcLocal,
			LocalPath: p,
		})
	}
	log.Println("local plugins list: ", list)
	return list
}

func SystemPluginSrcDefs() []sdkutils.PluginSrcDef {
	list := []sdkutils.PluginSrcDef{}
	paths := SearchPluginDirs(filepath.Join("plugins", "system"))
	for _, pluginPath := range paths {
		list = append(list, sdkutils.PluginSrcDef{
			Src:       sdkutils.PluginSrcSystem,
			LocalPath: pluginPath,
		})
	}
	log.Println("system plugins list: ", list)
	return list
}

func InstalledPluginsDef() []sdkutils.PluginSrcDef {
	list := []sdkutils.PluginSrcDef{}
	paths := InstalledPluginDirs()
	for _, p := range paths {
		info, err := sdkutils.GetPluginInfoFromPath(p)
		if err != nil {
			log.Println("Error reading plugin info: ", err)
			continue
		}
		metadata, err := ReadMetadata(info.Package)
		if err != nil {
			log.Println("Error reading plugin metadata: ", err)
			continue
		}

		if info.Package == metadata.Package {
			list = append(list, metadata.Def)
		}
	}
	return list
}

func SearchPluginDirs(searchPath string) (pluginDirs []string) {
	var list []string
	if err := sdkutils.FsListDirs(searchPath, &list, false); err != nil {
		log.Println("Error listing directories in ", searchPath, ": ", err)
		return
	}
	for _, p := range list {
		if err := ValidateSrcPath(p); err == nil {
			pluginDirs = append(pluginDirs, p)
		} else {
			fmt.Println("Error validating source path: ", p, err)
		}
	}
	return
}

// InstalledPluginDirs returns the list of installed plugins in the plugins directory. The path of each plugin is an aboslute path.
func InstalledPluginDirs() (pluginDirs []string) {
	installedPluginsPath := filepath.Join(sdkutils.PathPluginsDir, "installed")

	// check if plugins/installed directory exists before traversing
	if !(sdkutils.FsExists(installedPluginsPath)) {
		return
	}

	// this lists all directories inside paths.PluginsDir/installed
	var list []string
	if err := sdkutils.FsListDirs(installedPluginsPath, &list, false); err != nil {
		fmt.Printf("Error listing directories in %s: %v\n", installedPluginsPath, err)
		return
	}

	for _, p := range list {
		if err := ValidateInstallPath(p); err == nil {
			pluginDirs = append(pluginDirs, p)
		} else {
			fmt.Println("Error validating install path: ", p, err)
		}
	}

	return
}

func GetMetaDataPath(pkg string) string {
	return filepath.Join(sdkutils.PathConfigDir, "plugins", pkg, "metadata.json")
}

func WriteMetadata(def sdkutils.PluginSrcDef, pkg string) error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	meta := sdkutils.PluginMetadata{
		Package: pkg,
		Def:     def,
	}

	for i, m := range cfg.Metadata {
		if m.Package == pkg {
			cfg.Metadata[i] = meta
			return config.WritePluginsConfig(cfg)
		}
	}

	cfg.Metadata = append(cfg.Metadata, meta)

	return config.WritePluginsConfig(cfg)
}

func ReadMetadata(pkg string) (metadata sdkutils.PluginMetadata, err error) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return
	}

	for _, m := range cfg.Metadata {
		if m.Package == pkg {
			return m, nil
		}
	}

	return metadata, ErrNotInstalled
}

func IsPackageInstalled(pkg string) bool {
	installPath := GetInstallPath(pkg)
	err := ValidateInstallPath(installPath)
	return err == nil
}

func IsSrcDefInstalled(def sdkutils.PluginSrcDef) bool {
	installPath, ok := FindDefInstallPath(def)
	if !ok {
		return false
	}

	err := ValidateInstallPath(installPath)
	return err == nil
}

func InstalledPluginsList() (list []sdkutils.PluginMetadata) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return list
	}

	list = []sdkutils.PluginMetadata{}
	for _, m := range cfg.Metadata {
		if IsSrcDefInstalled(m.Def) {
			list = append(list, m)
		}
	}

	return
}

func NeedsRecompile(def sdkutils.PluginSrcDef) bool {
	if env.GO_ENV == env.ENV_DEV && (def.Src == sdkutils.PluginSrcLocal || def.Src == sdkutils.PluginSrcSystem) {
		return true
	}

	info, err := GetInfoFromDef(def)
	if err != nil {
		return true
	}

	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		log.Println("Error reading plugins config: ", err)
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
	if err := sdkutils.FsCopy(updatePath, GetInstallPath(pkg)); err != nil {
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
	if sdkutils.FsExists(backupPath) {
		if err := os.RemoveAll(backupPath); err != nil {
			return err
		}
	}
	return sdkutils.FsCopy(installPath, backupPath)
}

func HasBackup(pkg string) bool {
	backup := GetBackupPath(pkg)
	err := ValidateInstallPath(backup)
	return err == nil
}

func RestoreBackup(pkg string) error {
	backupPath := GetBackupPath(pkg)
	if err := sdkutils.FsCopy(backupPath, GetInstallPath(pkg)); err != nil {
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

func ValidateSrcPath(src string) error {
	requiredFiles := []string{"plugin.json", "go.mod", "main.go", "LICENSE.txt"}

	for _, f := range requiredFiles {
		if !sdkutils.FsExists(filepath.Join(src, f)) {
			return errors.New(f + " not found in source path")
		}
	}
	return nil
}

func ValidateInstallPath(src string) error {
	requiredFiles := []string{"plugin.json", "go.mod", "plugin.so"}

	for _, f := range requiredFiles {
		if !sdkutils.FsExists(filepath.Join(src, f)) {
			return errors.New(f + " not found in source path")
		}
	}
	return nil
}

func FindDefInstallPath(def sdkutils.PluginSrcDef) (installPath string, ok bool) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return
	}

	for _, meta := range cfg.Metadata {
		if def.Equal(meta.Def) {
			return GetInstallPath(meta.Package), true
		}
	}

	return "", false
}

func GetAuthorNameFromGitUrl(def sdkutils.PluginSrcDef) string {
	return strings.Split(strings.TrimPrefix(def.GitURL, "https://github.com/"), "/")[0]
}

func GetRepoFromGitUrl(def sdkutils.PluginSrcDef) string {
	return strings.Split(strings.TrimPrefix(def.GitURL, fmt.Sprintf("https://github.com/%s/", GetAuthorNameFromGitUrl(def))), "/")[0]
}

func GetInstallPath(pkg string) string {
	return filepath.Join(sdkutils.PathPluginsDir, "installed", pkg)
}

func GetPendingUpdatePath(pkg string) string {
	return filepath.Join(sdkutils.PathPluginsDir, "update", pkg)
}

func GetBackupPath(pkg string) string {
	return filepath.Join(sdkutils.PathPluginsDir, "backup", pkg)
}
