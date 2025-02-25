package plugins

import (
	"core/env"
	"core/internal/utils/encdisk"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

func BuildFromLocal(w io.Writer, db *pgxpool.Pool, def sdkutils.PluginSrcDef) (sdkutils.PluginInfo, error) {
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

func BuildFromGit(w io.Writer, db *pgxpool.Pool, def sdkutils.PluginSrcDef) (sdkutils.PluginInfo, error) {
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

	if err := sdkutils.GitClone(w, repo, clonepath); err != nil {
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

func BuildPluginSo(pluginSrcDir string, workdir string) error {
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

	buildpath := filepath.Join(workdir, "plugins", info.Package)

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

	if err := sdkutils.FsCopyDir(pluginSrcDir, buildpath, nil); err != nil {
		return err
	}

	if err := sdkutils.FsCopyDir(filepath.Join(sdkutils.PathAppDir, "sdk"), filepath.Join(workdir, "sdk"), nil); err != nil {
		return err
	}

	libs := []string{}
	err := sdkutils.FsListDirs("sdk/libs", &libs, false)
	if err != nil {
		return err
	}

	goWork := fmt.Sprintf(`
go %s

use (
    ./sdk/api
    ./sdk/utils
    `, sdkutils.GO_VERSION)

	for _, lib := range libs {
		goWork += fmt.Sprintf("./sdk/libs/%s\n", filepath.Base(lib))
	}

	goWork += fmt.Sprintf("./plugins/%s\n)", info.Package)
	goworkFile := filepath.Join(workdir, "go.work")
	if err := os.WriteFile(goworkFile, []byte(goWork), sdkutils.PermFile); err != nil {
		return err
	}

	if err := BuildAssets(pluginSrcDir); err != nil {
		return err
	}

	// Don't build templates in development since it is already watched and built by another script.
	if env.GO_ENV != env.ENV_DEV {
		if err := BuildTemplates(buildpath); err != nil {
			return err
		}
	}

	gofile := "main.go"
	outfile := "plugin.so"
	if err := BuildGoPlugin(gofile, outfile, buildpath, nil); err != nil {
		return err
	}

	pluginSoOut := filepath.Join(buildpath, "plugin.so")
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
		BuildTags: env.BuildTags,
		ExtraArgs: extraArgs,
	}

	if err := sdkutils.BuildGoModule(gofile, outfile, buildOpts); err != nil {
		return err
	}

	return nil
}
