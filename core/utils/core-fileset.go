package tools

import (
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// CoreFileSet is the set of core-layer paths (relative to the app root) that make
// up a non-mono install, excluding plugins/installed, the persistent data/ tree,
// the operator defaults/, and the built binaries (bin/flare, core/plugin.so). It is
// shared by:
//
//   - create-core-files (the full OS-image core files), and
//   - build-core-bins (the core_arch_bin UPDATE tarball downloaded during a
//     non-mono System Software update),
//
// so a core update overlays exactly the same core layer a fresh install ships and
// the two can never drift.
//
// plugins/installed is excluded so applying a core update never clobbers installed
// plugins. defaults/ is likewise EXCLUDED: the authoritative defaults come from the
// operator's data config zip (the software-release build's TidyConfigFiles copies
// data/config into defaults/, and the OS image ships that), NOT from the core repo's
// generic defaults/ (e.g. themes.json = com.flarego.core-theme). Shipping the repo
// defaults in the update tarball would overwrite the operator's config-zip defaults
// on every core update — so the core layer carries no defaults/ at all and the
// device keeps the defaults it was installed with.
//
// data/plugins/system is also EXCLUDED: system plugin Go is statically linked into
// core/plugin.so by sysplugin-prepare during the core_arch_bin build, and their
// runtime data tree is staged into plugins/installed/<pkg>. The source tree is
// therefore redundant on-device, so neither the fresh-install core files nor the
// core-update tarball carry it — matching the software-release packaging, which
// strips it after the core is built.
func CoreFileSet() []string {
	files := []string{
		"core/go.mod",
		"core/go.sum",
		"core/sqlc.yml",
		"core/package.json",
		"core/package-lock.json",
		"core/plugin.json",
		"core/resources",
		"core/utils",
		"sdk",
		"scripts",
		"go.work.default",
	}

	// Per-partner product version stamped by the software-release build. Ships in
	// both the fresh-install core layer and the core UPDATE tarball so a device
	// always reports its current product version after an update (unlike
	// /etc/os_release.json, which is flash-only). Included only when present:
	// unstamped trees (releases built before this existed, dev) omit it without
	// panicking the packager (BuildOutput errors on a missing listed file), and the
	// device then falls back to its core version via product.Version().
	if sdkutils.FsExists(filepath.Join(sdkutils.PathCoreDir, "product.json")) {
		files = append(files, "core/product.json")
	}

	return files
}

// CoreCustomFiles are the renamed copies both packagers emit: go.work (from
// go.work.default) and the build-appropriate start.sh (staged-overlay applier for
// non-mono via StartScriptSrc, wipe-and-restore for mono).
func CoreCustomFiles() []sdkutils.BuildOutputCustomEntry {
	return []sdkutils.BuildOutputCustomEntry{
		{Src: "go.work.default", Dest: "go.work"},
		{Src: StartScriptSrc(), Dest: "start.sh"},
	}
}
