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
}

func BuildLocalPlugins(opts BuildOpts) error {
	return BuildPluginDefs(LocalPluginSrcDefs(), opts)
}

func BuildSystemPlugins(opts BuildOpts) error {
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

	if err := BuildPluginSo(pluginPath, workdir, opts); err != nil {
		return err
	}

	if err := BuildAssets(pluginPath); err != nil {
		return err
	}

	return nil
}
