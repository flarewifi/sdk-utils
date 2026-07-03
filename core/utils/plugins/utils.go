package plugins

import (
	"core/utils/config"
	"core/utils/env"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"slices"

	sdkutils "github.com/flarewifi/sdk-utils"
)

var (
	ErrNotInstalled = errors.New("Plugin is not installed")
)

// CorePkg is the package id the core itself stages under in the unified update
// root (data/storage/system/updates/com.flarego.core). It MUST match CORE_PKG in
// start.sh, which overlays this package onto the app dir on the next boot.
const CorePkg = "com.flarego.core"

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

// DevelPluginSrcDefs lists the editable, rebuilt-from-source plugins under
// data/plugins/devel/. These are a development-only convenience and load ONLY in
// dev and devkit builds (both compile to GO_ENV == ENV_DEV). Staging and
// production ignore the devel directory entirely.
//
// This is the single authoritative gate, enforced for every caller:
//   - non-mono boot (boot.InitPlugins) — runtime: env.GO_ENV is baked into the
//     device binary, so a staging/prod device sees an empty list.
//   - mono generation (tools.MakePluginInitMono / mono-bin-prepare) — build time:
//     the prepare tool is compiled with the target's env build tag (derived from
//     the builder's RAILS_ENV), so devel plugins are never even statically linked
//     into a staging/prod mono binary, nor copied into plugins/installed.
//
// Returning nil here (rather than gating at each call site) means no current or
// future caller can accidentally pull devel plugins into a deployed build.
func DevelPluginSrcDefs() []sdkutils.PluginSrcDef {
	if !env.IsDevEnv() {
		return nil
	}
	list := []sdkutils.PluginSrcDef{}
	paths := SearchPluginDirs(sdkutils.PathPluginDevelDir)
	for _, pluginPath := range paths {
		list = append(list, sdkutils.PluginSrcDef{
			Src:       sdkutils.PluginSrcLocal,
			LocalPath: sdkutils.StripRootPath(pluginPath),
		})
	}
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
	return list
}

// InstalledLocalPluginSrcDirs returns the source directories of every INSTALLED,
// local-sourced plugin — the plugins that are recompiled on-device. Routing is by the
// installed metadata Def.Src (the source of truth: a store-registered plugin can have
// local source present, and a release bundles system sources under data/plugins/local
// too), NOT by where the source sits. Store plugins are rebuilt server-side and system
// plugins are compiled into the core image, so both are excluded here. Shared by the
// staged-update recompile (system-update) and the store-install reconcile (plugins-mgr).
func InstalledLocalPluginSrcDirs() ([]string, error) {
	var dirs []string
	for _, def := range LocalPluginSrcDefs() {
		srcDir, err := sdkutils.FindPluginSrc(def.LocalPath)
		if err != nil {
			continue
		}
		info, err := sdkutils.GetPluginInfoFromPath(srcDir)
		if err != nil {
			continue
		}
		// Skip plugins that are not installed (nothing to swap).
		if !sdkutils.FsExists(GetInstallPath(info.Package)) {
			continue
		}
		// Skip store- and system-sourced plugins. The store path owns the store
		// rebuild; system plugins ship inside the core and are never rebuilt here.
		if meta, err := ReadMetadata(info.Package); err == nil &&
			(meta.Def.Src == sdkutils.PluginSrcStore || meta.Def.Src == sdkutils.PluginSrcSystem) {
			continue
		}
		dirs = append(dirs, srcDir)
	}
	return dirs, nil
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
	return list
}

// ReconcileLocalDevelPluginsConfig makes data/config/plugins.json reflect the
// plugins physically present under data/plugins/local and data/plugins/devel —
// the authoritative location for local- and devel-sourced plugin sources. It
// returns the packages it added and removed (nil,nil when nothing changed).
//
//   - Add: a plugin directory found under local/ (always) or devel/ (dev/devkit
//     only, mirroring DevelPluginSrcDefs) that has no plugins.json entry is
//     registered as a Src=local, Standalone record — the same shape an operator
//     used to add by hand. Registration is keyed by the plugin's package id, so a
//     plugin already recorded under any source (e.g. a store entry with local
//     source present) is left as-is and never duplicated.
//   - Remove: an existing Src=local entry whose LocalPath points under local/ or
//     devel/ but whose directory no longer exists is dropped. The directory is
//     authoritative, so a removed source means a removed plugin.
//
// Only Src=local entries under those two dirs are ever touched. Store/git entries
// (empty LocalPath), system entries (LocalPath under the system dir), and meta
// records structurally cannot match either rule, so they are always preserved.
//
// Removal keys off real directory existence (FsExists), NOT the scan, so it is
// safe in every environment: a deployed device that does not scan the devel dir
// still won't strip an entry merely because it went unscanned.
func ReconcileLocalDevelPluginsConfig() (added, removed []string, err error) {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return nil, nil, err
	}

	changed := false

	// Prune stale local/devel entries whose source dir is gone.
	localPrefix := sdkutils.StripRootPath(sdkutils.PathPluginLocalDir) + "/"
	develPrefix := sdkutils.StripRootPath(sdkutils.PathPluginDevelDir) + "/"
	retained := make([]sdkutils.PluginMetadata, 0, len(cfg.Metadata))
	for _, m := range cfg.Metadata {
		if m.Def.Src == sdkutils.PluginSrcLocal {
			lp := m.Def.LocalPath
			underLocalDevel := strings.HasPrefix(lp, localPrefix) || strings.HasPrefix(lp, develPrefix)
			if underLocalDevel && !sdkutils.FsExists(filepath.Join(sdkutils.PathAppDir, lp)) {
				removed = append(removed, m.Package)
				changed = true
				continue
			}
		}
		retained = append(retained, m)
	}
	cfg.Metadata = retained

	// Register plugin dirs present on disk but missing from the config.
	dirs := SearchPluginDirs(sdkutils.PathPluginLocalDir)
	if env.IsDevEnv() {
		dirs = append(dirs, SearchPluginDirs(sdkutils.PathPluginDevelDir)...)
	}
	for _, dir := range dirs {
		info, infoErr := sdkutils.GetPluginInfoFromPath(dir)
		if infoErr != nil {
			continue
		}
		if metadataHasPkg(cfg.Metadata, info.Package) {
			continue
		}
		cfg.Metadata = append(cfg.Metadata, sdkutils.PluginMetadata{
			Package: info.Package,
			Def: sdkutils.PluginSrcDef{
				Src:       sdkutils.PluginSrcLocal,
				LocalPath: sdkutils.StripRootPath(dir),
			},
			Standalone: true,
		})
		added = append(added, info.Package)
		changed = true
	}

	if !changed {
		return nil, nil, nil
	}
	if err := config.WritePluginsConfig(cfg); err != nil {
		return nil, nil, err
	}
	return added, removed, nil
}

func metadataHasPkg(list []sdkutils.PluginMetadata, pkg string) bool {
	for _, m := range list {
		if m.Package == pkg {
			return true
		}
	}
	return false
}

func InstalledPluginsDef() []sdkutils.PluginSrcDef {
	list := []sdkutils.PluginSrcDef{}
	paths := InstalledPluginDirs()
	for _, p := range paths {
		info, err := sdkutils.GetPluginInfoFromPath(p)
		if err != nil {
			continue
		}
		metadata, err := ReadMetadata(info.Package)
		if err != nil {
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
		return
	}
	for _, p := range list {
		if err := ValidateSrcPath(p); err == nil {
			pluginDirs = append(pluginDirs, p)
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
		return
	}

	for _, p := range list {
		if err := ValidateInstallPath(p); err == nil {
			pluginDirs = append(pluginDirs, p)
		}
	}

	return
}

func GetMetaDataPath(pkg string) string {
	return filepath.Join(sdkutils.PathConfigDir, "plugins", pkg, "metadata.json")
}

// WriteMetadata records a standalone (user-initiated) install. It preserves an
// existing Standalone flag so that a plugin which is both a meta member and a
// standalone install keeps that signal.
func WriteMetadata(def sdkutils.PluginSrcDef, pkg string) error {
	return upsertMetadata(def, pkg, true)
}

// WriteMetadataAsMember records an install performed on behalf of a meta plugin.
// The plugin is recorded as a non-standalone install; meta ownership itself is
// tracked by the meta install record's member list, not here. An existing
// Standalone flag is preserved (a member the user also installed directly stays
// standalone).
func WriteMetadataAsMember(def sdkutils.PluginSrcDef, pkg string) error {
	return upsertMetadata(def, pkg, false)
}

// upsertMetadata writes or updates a plugin's metadata entry. When standalone is
// true the Standalone flag is set; an existing true flag is never cleared.
func upsertMetadata(def sdkutils.PluginSrcDef, pkg string, standalone bool) error {
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
			cfg.Metadata[i] = meta
			return config.WritePluginsConfig(cfg)
		}
	}

	cfg.Metadata = append(cfg.Metadata, meta)

	return config.WritePluginsConfig(cfg)
}

// WriteMetaRecord upserts a meta plugin install record (keyed by package).
func WriteMetaRecord(rec sdkutils.MetaPlugin) error {
	cfg, err := config.ReadPluginsConfig()
	if err != nil {
		return err
	}

	for i, m := range cfg.MetaPlugins {
		if m.Package == rec.Package {
			cfg.MetaPlugins[i] = rec
			return config.WritePluginsConfig(cfg)
		}
	}

	cfg.MetaPlugins = append(cfg.MetaPlugins, rec)
	return config.WritePluginsConfig(cfg)
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

func HasPendingUpdate(pkg string) bool {
	updatepath := GetPendingUpdatePath(pkg)
	return ValidateInstallPath(updatepath) == nil
}

// HasPendingCoreUpdate reports whether a staged core update exists under the
// unified update root. The core can't be detected with HasPendingUpdate: its
// self-contained tarball carries the manifest at core/plugin.json (not a root
// plugin.json), so we sentinel on the built CLI binary (bin/flare) instead, which
// only a fully-extracted core payload contains.
func HasPendingCoreUpdate() bool {
	return sdkutils.FsExists(filepath.Join(GetPendingUpdatePath(CorePkg), "bin", "flare"))
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

// GetPendingUpdatePath returns the unified staging directory for a package's
// downloaded-but-not-yet-applied update: data/storage/system/updates/{pkg}. BOTH
// the core (pkg == "com.flarego.core") and each plugin stage here; the non-mono
// boot script (start.sh) overlays every staged package onto its install
// location atomically on the next boot.
//
// Previously plugin updates staged under plugins/updates/{pkg} and were applied by
// the Go boot (MovePendingUpdate); applying is now the shell's job so the core and
// plugins follow one update process from a single staging root.
func GetPendingUpdatePath(pkg string) string {
	return filepath.Join(sdkutils.PathSystemUpdateDir, pkg)
}

// StagedCompleteMarkerPath returns the marker file the non-mono boot script waits
// for before applying staged updates. The app writes it only after the full set
// (core + any ABI-matched plugins) has finished staging, so a partial/aborted
// download is never applied.
func StagedCompleteMarkerPath() string {
	return filepath.Join(sdkutils.PathSystemUpdateDir, ".staged_complete")
}

// MarkStagedComplete commits the current staged set by writing the
// .staged_complete marker. It must be called LAST, only once every package in the
// set (core and/or plugins) has been fully extracted and validated — start.sh
// applies the whole staging root atomically the moment this marker exists.
func MarkStagedComplete() error {
	if err := sdkutils.FsEnsureDir(sdkutils.PathSystemUpdateDir); err != nil {
		return err
	}
	return os.WriteFile(StagedCompleteMarkerPath(), []byte("complete"), 0644)
}

func GetBackupPath(pkg string) string {
	return filepath.Join(sdkutils.PathPluginBackupsDir, pkg)
}
