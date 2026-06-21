package tools

import (
	"core/utils/plugins"
	"os"
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func BuildCoreBins(opts plugins.BuildOpts) {
	BuildFlareCLI()
	BuildCore(opts)

	// The non-mono core update is a SELF-CONTAINED tarball: the built binaries
	// (bin/flare, core/plugin.so — the latter with system plugins statically
	// linked) PLUS the full core layer (resources, sdk, scripts, module files,
	// start.sh, …). The non-mono start.sh overlays this onto the app dir in one
	// shot, so a core update refreshes everything a fresh install has EXCEPT the
	// operator-owned layers it must never clobber: plugins/installed, the
	// persistent data/ tree (including data/plugins/system, whose Go is already in
	// core/plugin.so), and defaults/ (the authoritative defaults come from the data
	// config zip via the software-release build, not the core repo) — all excluded
	// by CoreFileSet.
	build := &sdkutils.BuildOutput{
		OutputDir: filepath.Join(sdkutils.PathAppDir, "output/core-binaries"),
		Files:     append([]string{"bin/flare", "core/plugin.so"}, CoreFileSet()...),
		Custom:    CoreCustomFiles(),
	}

	if err := build.Run(); err != nil {
		panic(err)
	}
}

func BuildCore(opts plugins.BuildOpts) {

	if !opts.SkipTemplates {
		if err := plugins.BuildTemplates(sdkutils.PathCoreDir); err != nil {
			panic(err)
		}
	}

	if !opts.SkipQueries {
		if err := plugins.BuildQueries(sdkutils.PathCoreDir); err != nil {
			panic(err)
		}
	}

	if err := plugins.BuildAssets(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}

	workdir := filepath.Join(sdkutils.PathTmpDir, "b/core", sdkutils.RandomStr(16))
	defer os.RemoveAll(workdir)
	if err := plugins.BuildPluginSo(sdkutils.PathCoreDir, workdir, opts); err != nil {
		panic(err)
	}
}
