package tools

import (
	"fmt"
	"os"
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

const (
	// devkitPluginRepo is the GitHub repo (owner/name) the Devkit theme system
	// plugin is fetched from at devkit-build time. The plugin source no longer
	// lives committed under data/plugins/system — it is cloned here so the build
	// scanners (sysplugin-prepare, link_system_plugin_modules, SystemPluginSrcDefs)
	// can statically link it into core/plugin.so. Override with DEVKIT_PLUGIN_REPO.
	// The repo name matches the plugin package (com.flarego.devkit); the package is
	// still read from the cloned plugin.json, not inferred from the repo name.
	devkitPluginRepo = "flarewifi/com.flarego.devkit"

	// devkitPluginPkg is the canonical package/dir name the rest of the build and
	// the runtime themes.json default key off. We clone INTO this directory name
	// regardless of the repo name so every downstream reference resolves unchanged.
	devkitPluginPkg = "com.flarego.devkit"
)

// CloneDevkitPlugin fetches the Devkit theme system plugin from GitHub into
// data/plugins/system/<devkitPluginPkg>. It must run before any build step that
// scans data/plugins/system (sysplugin-prepare, link_system_plugin_modules,
// BuildCore's static link). Panics on failure — a devkit build with no Devkit
// theme is not a valid artifact.
func CloneDevkitPlugin() {
	repo := devkitPluginRepo
	if v := os.Getenv("DEVKIT_PLUGIN_REPO"); v != "" {
		repo = v
	}
	ref := os.Getenv("DEVKIT_PLUGIN_REF") // empty = default branch

	// The repo is public — clone over anonymous HTTPS, no credentials needed.
	// Disable git's interactive credential prompt anyway so a transient access
	// failure fails fast with a clear error instead of blocking on a terminal
	// read (which in Docker surfaces as the cryptic
	// "could not read Username ... No such device or address").
	os.Setenv("GIT_TERMINAL_PROMPT", "0")

	url := fmt.Sprintf("https://github.com/%s.git", repo)

	dest := filepath.Join(sdkutils.PathPluginSystemDir, devkitPluginPkg)

	// Start clean so a re-run never merges a stale tree over a fresh clone.
	if err := os.RemoveAll(dest); err != nil {
		panic(fmt.Errorf("clearing devkit plugin dir: %w", err))
	}
	if err := sdkutils.FsEnsureDir(sdkutils.PathPluginSystemDir); err != nil {
		panic(fmt.Errorf("creating system plugins dir: %w", err))
	}

	fmt.Printf("Cloning devkit plugin %s -> %s\n", repo, sdkutils.StripRootPath(dest))
	if err := sdkutils.GitClone(sdkutils.GitRepoSource{URL: url, Ref: ref}, dest); err != nil {
		panic(fmt.Errorf("cloning devkit plugin %s: %w", repo, err))
	}
}
