package plugins

import (
	"path/filepath"
	"sync"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// validityFlags mirrors the four on-disk marker files that withhold a plugin
// from the boot loader (see blockedMarker, disabledMarker, updateSkippedMarker,
// and the "uninstall" marker in plugin-install.go).
type validityFlags struct {
	blocked       bool
	disabled      bool
	updateSkipped bool
	toBeRemoved   bool
}

// validityMu guards validityCache. A plain mutex (not sync.Map) matches this
// codebase's idiom for a central shared registry -- see scheduler.Manager.
var (
	validityMu    sync.RWMutex
	validityCache = map[string]validityFlags{}
)

// LoadValidityCache seeds the in-memory validity registry from the on-disk
// markers of every installed plugin. It must run once at boot, before
// InitPlugins loads any plugin .so and before InitHttpServer starts accepting
// requests -- see boot.Init. Afterward, IsBlocked/IsDisabled/IsUpdateSkipped/
// IsToBeRemoved/IsInvalid are served from this cache instead of re-reading the
// filesystem on every HTTP request and mono-loader dispatch.
//
// This closes a runtime tampering window: a loaded plugin .so shares the host
// process's own filesystem privileges, so a compromised/malicious plugin could
// otherwise delete its own "blocked"/"disabled" marker file to bypass the
// cloud denylist or a lapsed store purchase without the process ever
// restarting. Once seeded, the cache is the sole source of truth for these
// checks at runtime; it only changes through refreshValidity, called by the
// marker writers in plugin-install.go right after each on-disk write.
func LoadValidityCache() {
	validityMu.Lock()
	defer validityMu.Unlock()
	validityCache = map[string]validityFlags{}
	for _, dir := range InstalledPluginDirs() {
		info, err := sdkutils.GetPluginInfoFromPath(dir)
		if err != nil {
			continue
		}
		validityCache[info.Package] = readValidityFlags(info.Package)
	}
}

// refreshValidity re-stats pkg's marker files and updates its in-memory entry.
// Called by every marker writer (BlockPlugin, UnblockPlugin, DisablePlugin,
// EnablePlugin, MarkUpdateSkipped, MarkToRemove) right after the on-disk write
// succeeds, and by InstallPlugin/InstallPrebuilt/UninstallPlugin so a freshly
// (re)installed or removed plugin's cache entry never lingers stale.
func refreshValidity(pkg string) {
	flags := readValidityFlags(pkg)
	validityMu.Lock()
	validityCache[pkg] = flags
	validityMu.Unlock()
}

// cachedValidity returns pkg's in-memory validity flags. A pkg with no entry
// (never seeded/refreshed) returns the zero value -- i.e. "valid" -- which is
// the correct default for a plugin with no marker files.
func cachedValidity(pkg string) validityFlags {
	validityMu.RLock()
	defer validityMu.RUnlock()
	return validityCache[pkg]
}

func readValidityFlags(pkg string) validityFlags {
	installPath := GetInstallPath(pkg)
	return validityFlags{
		blocked:       sdkutils.FsExists(filepath.Join(installPath, blockedMarker)),
		disabled:      sdkutils.FsExists(filepath.Join(installPath, disabledMarker)),
		updateSkipped: sdkutils.FsExists(filepath.Join(installPath, updateSkippedMarker)),
		toBeRemoved:   sdkutils.FsExists(filepath.Join(installPath, "uninstall")),
	}
}
