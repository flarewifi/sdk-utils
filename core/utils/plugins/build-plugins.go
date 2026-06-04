package plugins

import (
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// BuildOpts controls whether codegen steps run during a build. The sqlc and
// templ outputs are committed to the repo, so builders that run where those
// tools are unavailable (e.g. on-device core_arch_bin builds) can skip
// generation and rely on the committed files. The zero value generates
// everything, preserving the default behavior for callers that don't opt out.
type BuildOpts struct {
	SkipTemplates bool // skip `templ generate` (use committed *_templ.go)
	SkipQueries   bool // skip sqlc generation (use committed db/queries)
	// AppDir overrides the app root that BuildPluginSo copies core/, sdk/ and
	// scripts/ from when assembling the isolated build workspace. The zero value
	// uses the live install (sdkutils.PathAppDir). Set it to a STAGED core payload
	// (e.g. data/storage/system/updates/com.flarego.core) to compile a plugin .so
	// against a not-yet-applied core — used when recompiling local plugins during a
	// staged update so the .so is ABI-matched to the core it will boot with.
	AppDir string
	// SkipPluginSo stages a plugin WITHOUT compiling a standalone plugin.so.
	// Statically-linked system plugins are compiled into core/plugin.so by
	// sysplugin-prepare (see core/internal/api/system-plugins-init.go) and are
	// registered at boot by LoadSystemPlugins — the installed-dir loop then
	// SKIPS them (init-plugins.go), so a per-plugin .so would never be
	// dlopened. They still need their data tree on disk under plugins/installed,
	// so when this is set BuildPluginDefs stages the bundled set
	// (CopyPluginFilesMono — same as a mono build) instead of the full non-mono
	// set (CopyPluginFiles, which includes a standalone plugin.so).
	SkipPluginSo bool
}

func BuildLocalPlugins(opts BuildOpts) error {
	return BuildPluginDefs(LocalPluginSrcDefs(), opts)
}

func BuildSystemPlugins(opts BuildOpts) error {
	// System plugins are statically linked into core/plugin.so; never build a
	// throwaway standalone .so for them. BuildPluginDefs still stages their
	// data tree under plugins/installed (assets, migrations, plugin.json).
	opts.SkipPluginSo = true
	return BuildPluginDefs(SystemPluginSrcDefs(), opts)
}

func BuildPluginDefs(pluginDefs []sdkutils.PluginSrcDef, opts BuildOpts) error {
	for _, def := range pluginDefs {
		pluginPath, err := sdkutils.FindPluginSrc(def.LocalPath)
		if err != nil {
			return err
		}

		if err := BuildPlugin(pluginPath, opts); err != nil {
			return err
		}

		info, err := sdkutils.GetPluginInfoFromPath(pluginPath)
		if err != nil {
			return err
		}

		pluginInstallDir := filepath.Join(sdkutils.PathPluginInstallDir, info.Package)

		if err := os.RemoveAll(pluginInstallDir); err != nil {
			return err
		}

		// A statically-linked system plugin (SkipPluginSo) has its Go code in
		// core/plugin.so, so — exactly like a mono build — it gets the bundled
		// file set (PluginFilesMono: no plugin.so, no Go build inputs). A normal
		// non-mono plugin gets the full set (PluginFiles) including its .so.
		if opts.SkipPluginSo {
			if err := sdkutils.CopyPluginFilesMono(pluginPath, pluginInstallDir); err != nil {
				return err
			}
			continue
		}

		if err := sdkutils.CopyPluginFiles(pluginPath, pluginInstallDir); err != nil {
			return err
		}
	}
	return nil
}

func BuildPlugin(pluginPath string, opts BuildOpts) error {
	workdir := filepath.Join(sdkutils.PathTmpDir, "builds", filepath.Base(pluginPath))
	defer os.RemoveAll(workdir)

	if err := PatchPluginDeps(pluginPath); err != nil {
		return err
	}

	if !opts.SkipTemplates {
		if err := BuildTemplates(pluginPath); err != nil {
			return err
		}
	}

	if !opts.SkipQueries {
		if err := BuildQueries(pluginPath); err != nil {
			return err
		}
	}

	// Statically-linked system plugins are compiled into core/plugin.so, so they
	// need no standalone .so (boot never dlopens it). Skip the compile but still
	// generate templ/queries/assets so the staged data tree is complete.
	if !opts.SkipPluginSo {
		if err := BuildPluginSo(pluginPath, workdir, opts); err != nil {
			return err
		}
	}

	if err := BuildAssets(pluginPath); err != nil {
		return err
	}

	return nil
}
