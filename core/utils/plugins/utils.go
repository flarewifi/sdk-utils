package plugins

import (
	"core/utils/config"
	"core/utils/env"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"slices"

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
	localPlugins := LocalPluginSrcDefs()
	systemPlugins := SystemPluginSrcDefs()
	configPlugins := ConfigPluginSrcDefs()
	alldefs := append(systemPlugins, localPlugins...)
	alldefs = append(alldefs, configPlugins...)
	return alldefs
}

func ConfigPluginSrcDefs() []sdkutils.PluginSrcDef {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return nil
	}

	excluded := []string{sdkutils.PluginSrcLocal, sdkutils.PluginSrcSystem}
	var defs []sdkutils.PluginSrcDef
	for _, m := range cfg.Metadata {
		if slices.Contains(excluded, m.Def.Src) {
			continue
		}

		defs = append(defs, m.Def)
	}

	return defs
}

func ConfigPluginSrcDefsWithPkg() []sdkutils.PluginMetadata {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return nil
	}

	excluded := []string{sdkutils.PluginSrcLocal, sdkutils.PluginSrcSystem}
	var metadataList []sdkutils.PluginMetadata
	for _, m := range cfg.Metadata {
		if slices.Contains(excluded, m.Def.Src) {
			continue
		}

		metadataList = append(metadataList, m)
	}

	return metadataList
}

func DevelPluginSrcDefs() []sdkutils.PluginSrcDef {
	list := []sdkutils.PluginSrcDef{}
	paths := SearchPluginDirs(sdkutils.PathPluginDevelDir)
	for _, pluginPath := range paths {
		list = append(list, sdkutils.PluginSrcDef{
			Src:       sdkutils.PluginSrcLocal,
			LocalPath: sdkutils.StripRootPath(pluginPath),
		})
	}
	log.Println("devel plugins list: ", list)
	return list
}

func LocalPluginSrcDefs() []sdkutils.PluginSrcDef {
	list := []sdkutils.PluginSrcDef{}
	paths := SearchPluginDirs(sdkutils.PathPluginLocalDir)
	for _, pluginPath := range paths {
		list = append(list, sdkutils.PluginSrcDef{
			Src:       sdkutils.PluginSrcLocal,
			LocalPath: sdkutils.StripRootPath(pluginPath),
		})
	}
	log.Println("local plugins list: ", list)
	return list
}

func SystemPluginSrcDefs() []sdkutils.PluginSrcDef {
	list := []sdkutils.PluginSrcDef{}
	paths := SearchPluginDirs(sdkutils.PathPluginSystemDir)
	for _, pluginPath := range paths {
		list = append(list, sdkutils.PluginSrcDef{
			Src:       sdkutils.PluginSrcSystem,
			LocalPath: sdkutils.StripRootPath(pluginPath),
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

// InstalledPluginDirs returns the list of installed plugins in the plugins directory. The path of each plugin is an absolute path.
func InstalledPluginDirs() (pluginDirs []string) {
	// check if plugins/installed directory exists before traversing
	if !(sdkutils.FsExists(sdkutils.PathPluginInstallDir)) {
		return
	}

	// this lists all directories inside paths.PluginsDir/installed
	var list []string
	if err := sdkutils.FsListDirs(sdkutils.PathPluginInstallDir, &list, false); err != nil {
		fmt.Printf("Error listing directories in %s: %v\n", sdkutils.PathPluginInstallDir, err)
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

// WriteMetadata records a standalone (user-initiated) install. It preserves any
// existing meta ownership so that a plugin which is both a meta member and a
// standalone install keeps both signals.
func WriteMetadata(def sdkutils.PluginSrcDef, pkg string) error {
	return upsertMetadata(def, pkg, true, "")
}

// WriteMetadataAsMember records an install performed on behalf of a meta plugin.
// It appends metaPkg to the member's owners and preserves the existing
// Standalone flag (a member the user also installed directly stays standalone).
func WriteMetadataAsMember(def sdkutils.PluginSrcDef, pkg string, metaPkg string) error {
	return upsertMetadata(def, pkg, false, metaPkg)
}

// upsertMetadata writes or updates a plugin's metadata entry, merging ownership
// fields with any existing entry. When standalone is true the Standalone flag is
// set. When addOwner is non-empty it is appended to MetaOwners (deduped).
func upsertMetadata(def sdkutils.PluginSrcDef, pkg string, standalone bool, addOwner string) error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	if def.Src == sdkutils.PluginSrcGit || def.Src == sdkutils.PluginSrcStore {
		def.LocalPath = ""
	}

	meta := sdkutils.PluginMetadata{
		Package:    pkg,
		Def:        def,
		Standalone: standalone,
	}

	for i, m := range cfg.Metadata {
		if m.Package == pkg {
			meta.Standalone = m.Standalone || standalone
			meta.MetaOwners = m.MetaOwners
			if addOwner != "" && !slices.Contains(meta.MetaOwners, addOwner) {
				meta.MetaOwners = append(meta.MetaOwners, addOwner)
			}
			cfg.Metadata[i] = meta
			return config.WritePluginsConfig(cfg)
		}
	}

	if addOwner != "" {
		meta.MetaOwners = []string{addOwner}
	}
	cfg.Metadata = append(cfg.Metadata, meta)

	return config.WritePluginsConfig(cfg)
}

// AddMetaOwner records that metaPkg owns memberPkg without reinstalling it.
// Used when a member is already installed at an acceptable version.
func AddMetaOwner(memberPkg string, metaPkg string) error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	for i, m := range cfg.Metadata {
		if m.Package == memberPkg {
			if !slices.Contains(m.MetaOwners, metaPkg) {
				cfg.Metadata[i].MetaOwners = append(m.MetaOwners, metaPkg)
			}
			return config.WritePluginsConfig(cfg)
		}
	}

	return ErrNotInstalled
}

// RemoveMetaOwner drops metaPkg from memberPkg's owners and reports the member's
// remaining owner count and Standalone flag, so the caller can decide whether to
// uninstall the member. A member not found is treated as already gone (0, false).
func RemoveMetaOwner(memberPkg string, metaPkg string) (remaining int, standalone bool, err error) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return 0, false, err
	}

	for i, m := range cfg.Metadata {
		if m.Package == memberPkg {
			owners := slices.DeleteFunc(m.MetaOwners, func(o string) bool { return o == metaPkg })
			cfg.Metadata[i].MetaOwners = owners
			if err := config.WritePluginsConfig(cfg); err != nil {
				return 0, false, err
			}
			return len(owners), m.Standalone, nil
		}
	}

	return 0, false, nil
}

// WriteMetaRecord upserts a meta plugin install record (keyed by package).
func WriteMetaRecord(rec sdkutils.MetaInstallRecord) error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	for i, m := range cfg.Metas {
		if m.Package == rec.Package {
			cfg.Metas[i] = rec
			return config.WritePluginsConfig(cfg)
		}
	}

	cfg.Metas = append(cfg.Metas, rec)
	return config.WritePluginsConfig(cfg)
}

// ReadMetaRecord returns the install record for a meta plugin.
func ReadMetaRecord(pkg string) (sdkutils.MetaInstallRecord, error) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return sdkutils.MetaInstallRecord{}, err
	}

	for _, m := range cfg.Metas {
		if m.Package == pkg {
			return m, nil
		}
	}

	return sdkutils.MetaInstallRecord{}, ErrNotInstalled
}

// RemoveMetaRecord deletes a meta plugin install record.
func RemoveMetaRecord(pkg string) error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	for i, m := range cfg.Metas {
		if m.Package == pkg {
			cfg.Metas = slices.Delete(cfg.Metas, i, i+1)
			break
		}
	}

	return config.WritePluginsConfig(cfg)
}

// AllMetaRecords returns every installed meta plugin record.
func AllMetaRecords() ([]sdkutils.MetaInstallRecord, error) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return nil, err
	}
	return cfg.Metas, nil
}

func CacheAndRegisterPlugin(def sdkutils.PluginSrcDef, pkg string) error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}
	for _, m := range cfg.Metadata {
		if m.Package == pkg && m.Def.Src == sdkutils.PluginSrcStore {
			return nil // preserve marketplace entry
		}
	}
	// Write original def directly — no cache copy needed
	return WriteMetadata(def, pkg)
}

func ReadMetadata(pkg string) (metadata sdkutils.PluginMetadata, err error) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return metadata, err
	}

	for _, m := range cfg.Metadata {
		if m.Package == pkg {
			return m, nil
		}
	}

	return metadata, ErrNotInstalled
}

func RemoveMetadata(pkg string) error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	// Remove metadata from the list of installed plugins
	for i, m := range cfg.Metadata {
		if m.Package == pkg {
			cfg.Metadata = slices.Delete(cfg.Metadata, i, i+1)
			break
		}
	}

	return config.WritePluginsConfig(cfg)
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

	return slices.Contains(cfg.Recompile, info.Package)
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

// ValidateInstallPath checks whether a directory under plugins/installed/ holds
// a real plugin's data tree. Only plugin.json is required — plugin.so is NOT,
// because statically-linked system plugins (compiled into the core binary by
// sysplugin-prepare) carry their data tree here without a .so sibling. The
// previous plugin.so requirement was a leftover from the pre-static-plugin
// world and silently excluded system plugin dirs from InstalledPluginDirs(),
// which in turn skipped their migrations at boot in production.
func ValidateInstallPath(src string) error {
	requiredFiles := []string{"plugin.json"}

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
	return filepath.Join(sdkutils.PathPluginInstallDir, pkg)
}

func GetPendingUpdatePath(pkg string) string {
	return filepath.Join(sdkutils.PathPluginUpdatesDir, pkg)
}

func GetBackupPath(pkg string) string {
	return filepath.Join(sdkutils.PathPluginBackupsDir, pkg)
}

