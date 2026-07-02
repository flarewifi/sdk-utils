package plugins

import (
	"core/utils/env"
	"core/utils/tags"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func BuildFromLocal(w io.Writer, db *sql.DB, def sdkutils.PluginSrcDef) (sdkutils.PluginInfo, error) {
	err := InstallPlugin(def.LocalPath, db, InstallOpts{Def: def, RemoveSrc: false})
	if err != nil {
		return sdkutils.PluginInfo{}, err
	}

	info, err := GetInfoFromDef(def)
	if err != nil {
		return sdkutils.PluginInfo{}, err
	}

	if err := WriteMetadata(def, info.Package); err != nil {
		return sdkutils.PluginInfo{}, err
	}

	return info, nil
}

func BuildPluginSo(pluginSrcDir string, workdir string, opts BuildOpts) error {
	if pluginSrcDir == "" {
		return errors.New("Build plugin error: no plugin source path")
	}

	if workdir == "" {
		return errors.New("Build plugin error: no build path")
	}

	var info sdkutils.PluginInfo

	if err := sdkutils.JsonRead(filepath.Join(pluginSrcDir, "plugin.json"), &info); err != nil {
		return err
	}

	goBuildPath := filepath.Join(workdir, "plugins", info.Package)

	pluginSoPath := filepath.Join(pluginSrcDir, "plugin.so")
	os.Remove(pluginSoPath)

	if sdkutils.FsExists(pluginSoPath) {
		if err := os.Remove(pluginSoPath); err != nil {
			return err
		}
	}

	if err := sdkutils.FsEmptyDir(workdir); err != nil {
		return err
	}

	if err := sdkutils.FsEnsureDir(filepath.Join(workdir, "plugins")); err != nil {
		return err
	}

	// appDir is the source of the core/, sdk/ and scripts/ layers copied into the
	// build workspace. Defaults to the live install; opts.AppDir points it at a
	// staged core payload so a plugin can be compiled against a not-yet-applied core.
	appDir := sdkutils.PathAppDir
	if opts.AppDir != "" {
		appDir = opts.AppDir
	}

	if err := sdkutils.FsCopyDir(pluginSrcDir, goBuildPath, nil); err != nil {
		return err
	}

	if err := sdkutils.FsCopyDir(filepath.Join(appDir, "sdk"), filepath.Join(workdir, "sdk"), nil); err != nil {
		return err
	}

	// When building the core itself as a plugin, the plugin copy at
	// workdir/plugins/<package> IS the core module. Skipping the extra
	// ./core copy avoids two workspace entries declaring `module core`.
	isCoreBuild := filepath.Clean(pluginSrcDir) == filepath.Clean(sdkutils.PathCoreDir)

	// In a devkit build the core is closed-source: only the compiled core/plugin.so
	// and core/resources ship — no core Go source. 3rd-party plugins compile
	// against the SDK alone (sdk/api has zero `core` dependency), so skip copying
	// and `use`-ing ./core. Without this, runtime plugin builds would require core
	// source that the devkit deliberately omits.
	copyCore := !isCoreBuild && !tags.IsDevkit()
	if copyCore {
		if err := sdkutils.FsCopyDir(filepath.Join(appDir, "core"), filepath.Join(workdir, "core"), nil); err != nil {
			return err
		}
	}

	// When building the core, statically link the system plugins into
	// core/plugin.so. The generated core/internal/api/system-plugins-init.go
	// imports each system plugin's /system subpackage, so the plugin source must
	// live in this isolated workspace and be listed in go.work — otherwise those
	// imports don't resolve and the plugins aren't compiled into the .so. The
	// sources are copied as-is (sysplugin-prepare already generated their
	// system/main.go and, off-device, their templ/sqlc output); this mirrors how
	// CreateGoWorkspace adds data/plugins/system/* to the full workspace go.work.
	var systemPluginUsePaths []string
	if isCoreBuild {
		for _, def := range SystemPluginSrcDefs() {
			rel := filepath.ToSlash(def.LocalPath) // e.g. data/plugins/system/<name>
			srcAbs := filepath.Join(sdkutils.PathAppDir, def.LocalPath)
			dstAbs := filepath.Join(workdir, def.LocalPath)
			if err := sdkutils.FsCopyDir(srcAbs, dstAbs, nil); err != nil {
				return fmt.Errorf("copying system plugin %s into core build workspace: %w", rel, err)
			}
			systemPluginUsePaths = append(systemPluginUsePaths, "./"+rel)

			// In a devkit build, core/go.mod carries a require+replace for each
			// statically-linked system plugin (added in-place by make-devkit for
			// the flare CLI build, where the replace is relative to the repo-root
			// core/: ../data/plugins/system/<n>). Here core/go.mod has been copied
			// to workdir/plugins/core, so that relative path is wrong. Rewrite it to
			// ../../data/plugins/system/<n>, which matches the `use ./<rel>` written
			// to this workspace's go.work below. Under -trimpath a bare require
			// resolved only via `use` still fetches; the replace path must line up
			// with the use path string for the /system subpackage to resolve.
			if tags.IsDevkit() {
				modPath, err := readGoModModulePath(srcAbs)
				if err != nil {
					return fmt.Errorf("reading module path for system plugin %s: %w", rel, err)
				}
				if err := goModEditReplace(goBuildPath, modPath, "../../"+rel); err != nil {
					return err
				}
			}
		}
	}

	if err := sdkutils.FsCopyDir(filepath.Join(appDir, "scripts"), filepath.Join(workdir, "scripts"), nil); err != nil {
		return err
	}

	goWork := fmt.Sprintf(`
go %s

use (
    ./sdk/api
    ./sdk/utils
`, sdkutils.GO_VERSION)

	if copyCore {
		goWork += "    ./core\n"
	}
	goWork += fmt.Sprintf("    ./plugins/%s\n", info.Package)
	for _, p := range systemPluginUsePaths {
		goWork += "    " + p + "\n"
	}
	goWork += ")"
	goworkFile := filepath.Join(workdir, "go.work")
	if err := os.WriteFile(goworkFile, []byte(goWork), sdkutils.PermFile); err != nil {
		return err
	}

	// Don't build templates in development since it is already watched and built by another script.
	// Also skip when the caller opts out (e.g. on-device builds that rely on committed *_templ.go).
	if env.GO_ENV != env.ENV_DEV && !opts.SkipTemplates {
		if err := BuildTemplates(goBuildPath); err != nil {
			return err
		}
	}

	gofile := "."
	outfile := "plugin.so"
	if err := BuildGoPlugin(gofile, outfile, goBuildPath, nil); err != nil {
		return err
	}

	pluginSoOut := filepath.Join(goBuildPath, "plugin.so")
	if err := sdkutils.FsCopyFile(pluginSoOut, pluginSoPath); err != nil {
		return err
	}

	return nil
}

func BuildGoPlugin(gofile string, outfile string, workdir string, envs []string) error {
	goBin := GoBin()
	extraArgs := []string{"-buildmode=plugin"}

	// Route Go's build cache and link-temp onto the app data partition
	// (PathTmpDir == APP_TMP == /opt/flarewifi/tmp on-device). Without this,
	// `go build` falls back to $HOME/.cache/go-build and /tmp — on OpenWRT the
	// tiny rootfs overlay and a RAM-backed tmpfs — and linking a plugin.so against
	// a full core exhausts them ("no space left on device"). Pointing both at the
	// roomy data partition fixes that and lets successive plugin builds in a single
	// software-update run share one cache, speeding up the on-device recompile.
	goCacheDir := filepath.Join(sdkutils.PathCacheDir, "go-build")
	goTmpDir := filepath.Join(sdkutils.PathTmpDir, "go-tmp")
	if err := sdkutils.FsEnsureDir(goCacheDir); err != nil {
		return fmt.Errorf("create go build cache dir: %w", err)
	}
	if err := sdkutils.FsEnsureDir(goTmpDir); err != nil {
		return fmt.Errorf("create go tmp dir: %w", err)
	}
	envs = append(envs, "GOCACHE="+goCacheDir, "GOTMPDIR="+goTmpDir)

	buildOpts := sdkutils.GoBuildOpts{
		GoBinPath: goBin,
		WorkDir:   workdir,
		Env:       envs,
		BuildTags: tags.GetBuildTags(),
		ExtraArgs: extraArgs,
	}

	if err := sdkutils.BuildGoModule(gofile, outfile, buildOpts); err != nil {
		return err
	}

	return nil
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// readGoModModulePath returns the module path from the go.mod in goModDir.
func readGoModModulePath(goModDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(goModDir, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("no module directive in %s/go.mod", goModDir)
}

// goModEditReplace sets a path replace for modPath in the go.mod located at
// moduleDir (overwriting any existing replace for that module).
func goModEditReplace(moduleDir, modPath, replPath string) error {
	cmd := exec.Command(GoBin(), "mod", "edit", "-replace="+modPath+"="+replPath)
	cmd.Dir = moduleDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod edit -replace %s=%s in %s: %w\n%s", modPath, replPath, moduleDir, err, out)
	}
	return nil
}
