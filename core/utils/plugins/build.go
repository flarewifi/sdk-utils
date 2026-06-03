package plugins

import (
	"core/utils/encdisk"
	"core/utils/env"
	"core/utils/tags"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
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

	// TODO: remove logs
	log.Println("Marking plugins..")
	if err := WriteMetadata(def, info.Package); err != nil {
		return sdkutils.PluginInfo{}, err
	}

	return info, nil
}

func BuildFromGit(w io.Writer, db *sql.DB, def sdkutils.PluginSrcDef) (sdkutils.PluginInfo, error) {
	dev := sdkutils.Slugify(sdkutils.RandomStr(16), "_")
	parentpath := RandomPluginPath()
	diskfile := filepath.Join(parentpath, "plugin-clone", "disk", dev)
	mountpath := filepath.Join(parentpath, "plugin-build", "mount", dev)
	clonepath := filepath.Join(mountpath, "clone")
	mnt := encdisk.NewEncrypedDisk(diskfile, mountpath, dev)
	if err := mnt.Mount(); err != nil {
		return sdkutils.PluginInfo{}, err
	}

	w.Write([]byte("Cloning plugin from git: " + def.GitURL))
	repo := sdkutils.GitRepoSource{URL: def.GitURL, Ref: def.GitRef}

	if err := sdkutils.GitClone(repo, clonepath); err != nil {
		return sdkutils.PluginInfo{}, err
	}

	if err := InstallPlugin(clonepath, db, InstallOpts{Def: def}); err != nil {
		return sdkutils.PluginInfo{}, err
	}

	if err := mnt.Unmount(); err != nil {
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

	if err := sdkutils.FsCopyDir(pluginSrcDir, goBuildPath, nil); err != nil {
		return err
	}

	if err := sdkutils.FsCopyDir(filepath.Join(sdkutils.PathAppDir, "sdk"), filepath.Join(workdir, "sdk"), nil); err != nil {
		return err
	}

	// When building the core itself as a plugin, the plugin copy at
	// workdir/plugins/<package> IS the core module. Skipping the extra
	// ./core copy avoids two workspace entries declaring `module core`.
	isCoreBuild := filepath.Clean(pluginSrcDir) == filepath.Clean(sdkutils.PathCoreDir)
	if !isCoreBuild {
		if err := sdkutils.FsCopyDir(filepath.Join(sdkutils.PathAppDir, "core"), filepath.Join(workdir, "core"), nil); err != nil {
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
	// CreateGoWorkspace adds plugins/system/* to the full workspace go.work.
	var systemPluginUsePaths []string
	if isCoreBuild {
		for _, def := range SystemPluginSrcDefs() {
			rel := filepath.ToSlash(def.LocalPath) // e.g. plugins/system/<name>
			srcAbs := filepath.Join(sdkutils.PathAppDir, def.LocalPath)
			dstAbs := filepath.Join(workdir, def.LocalPath)
			if err := sdkutils.FsCopyDir(srcAbs, dstAbs, nil); err != nil {
				return fmt.Errorf("copying system plugin %s into core build workspace: %w", rel, err)
			}
			systemPluginUsePaths = append(systemPluginUsePaths, "./"+rel)
		}
	}

	if err := sdkutils.FsCopyDir(filepath.Join(sdkutils.PathAppDir, "scripts"), filepath.Join(workdir, "scripts"), nil); err != nil {
		return err
	}

	goWork := fmt.Sprintf(`
go %s

use (
    ./sdk/api
    ./sdk/utils
`, sdkutils.GO_VERSION)

	if !isCoreBuild {
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
	fmt.Printf("Copying '%s' to '%s'\n", sdkutils.StripRootPath(pluginSoOut), sdkutils.StripRootPath(pluginSoPath))
	if err := sdkutils.FsCopyFile(pluginSoOut, pluginSoPath); err != nil {
		log.Printf("Error copying '%s' to '%s': %+v\n", pluginSoOut, pluginSoPath, err)
		return err
	}

	return nil
}

func BuildGoPlugin(gofile string, outfile string, workdir string, envs []string) error {
	goBin := GoBin()
	extraArgs := []string{"-buildmode=plugin"}

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
