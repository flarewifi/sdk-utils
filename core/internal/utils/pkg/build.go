package pkg

import (
	"core/env"
	"core/internal/utils/cmd"
	"core/internal/utils/encdisk"
	"core/internal/utils/git"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	sdkplugin "sdk/api/plugin"

	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
	sdkruntime "github.com/flarehotspot/go-utils/runtime"
	sdkstr "github.com/flarehotspot/go-utils/strings"
)

func BuildFromLocal(w io.Writer, def PluginSrcDef) (sdkplugin.PluginInfo, error) {
	err := InstallPlugin(def.LocalPath, InstallOpts{RemoveSrc: false})
	if err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	info, err := GetPluginInfo(def)
	if err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	// TODO: remove logs
	log.Println("Marking plugins..")
	if err := WriteMetadata(def, GetInstallPath(info.Package)); err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	return info, nil
}

func BuildFromGit(w io.Writer, def PluginSrcDef) (sdkplugin.PluginInfo, error) {
	dev := sdkstr.Slugify(sdkstr.Rand(16), "_")
	parentpath := RandomPluginPath()
	diskfile := filepath.Join(parentpath, "plugin-clone", "disk", dev)
	mountpath := filepath.Join(parentpath, "plugin-build", "mount", dev)
	clonepath := filepath.Join(mountpath, "clone")
	mnt := encdisk.NewEncrypedDisk(diskfile, mountpath, dev)
	if err := mnt.Mount(); err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	w.Write([]byte("Cloning plugin from git: " + def.GitURL))
	repo := git.RepoSource{URL: def.GitURL, Ref: def.GitRef}

	if err := git.Clone(w, repo, clonepath); err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	if err := InstallPlugin(clonepath, InstallOpts{}); err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	if err := mnt.Unmount(); err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	info, err := GetPluginInfo(def)
	if err != nil {
		return sdkplugin.PluginInfo{}, err
	}

	installPath := GetInstallPath(info.Package)
	if err := WriteMetadata(def, installPath); err != nil {
		return sdkplugin.PluginInfo{}, err
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

	var info sdkplugin.PluginInfo

	pluginSoPath := filepath.Join(pluginSrcDir, "plugin.so")
	if err := sdkfs.ReadJson(filepath.Join(pluginSrcDir, "plugin.json"), &info); err != nil {
		return err
	}

	buildpath := filepath.Join(workdir, "plugins", info.Package)

	if sdkfs.Exists(pluginSoPath) {
		if err := os.Remove(pluginSoPath); err != nil {
			return err
		}
	}

	if err := sdkfs.EmptyDir(workdir); err != nil {
		return err
	}

	if err := sdkfs.EnsureDir(filepath.Join(workdir, "plugins")); err != nil {
		return err
	}

	if err := sdkfs.CopyDir(pluginSrcDir, buildpath, nil); err != nil {
		return err
	}

	if err := sdkfs.CopyDir(filepath.Join(sdkpaths.AppDir, "sdk"), filepath.Join(workdir, "sdk"), nil); err != nil {
		return err
	}

	libs := []string{}
	err := sdkfs.LsDirs("sdk/libs", &libs, false)
	if err != nil {
		return err
	}

	goWork := fmt.Sprintf(`
go %s

use (
    ./sdk/api
    ./sdk/utils
    `, sdkruntime.GO_VERSION)

	for _, lib := range libs {
		goWork += fmt.Sprintf("./sdk/libs/%s\n", filepath.Base(lib))
	}

	goWork += fmt.Sprintf("./plugins/%s\n)", info.Package)
	goworkFile := filepath.Join(workdir, "go.work")
	if err := os.WriteFile(goworkFile, []byte(goWork), sdkfs.PermFile); err != nil {
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
	args := &GoBuildArgs{WorkDir: buildpath, ExtraArgs: []string{"-buildmode=plugin"}}
	if err := BuildGoModule(gofile, outfile, args); err != nil {
		return err
	}

	pluginSoOut := filepath.Join(buildpath, "plugin.so")
	fmt.Printf("Copying '%s' to '%s'\n", sdkpaths.StripRoot(pluginSoOut), sdkpaths.StripRoot(pluginSoPath))
	if err := sdkfs.CopyFile(pluginSoOut, pluginSoPath); err != nil {
		log.Printf("Error copying '%s' to '%s': %+v\n", pluginSoOut, pluginSoPath, err)
		return err
	}

	return nil
}

type GoBuildArgs struct {
	WorkDir   string
	Env       []string
	ExtraArgs []string
}

func BuildGoModule(gofile string, outfile string, params *GoBuildArgs) error {
	if params == nil {
		params = &GoBuildArgs{}
	}

	fmt.Println("Building go module: " + sdkpaths.StripRoot(filepath.Join(params.WorkDir, gofile)))

	goBin := GoBin()
	buildArgs := BuildArgs()
	buildArgs = append(buildArgs, params.ExtraArgs...)

	buildCmd := []string{"build"}
	buildCmd = append(buildCmd, buildArgs...)
	buildCmd = append(buildCmd, "-o", outfile, gofile)

	cmdstr := goBin
	for _, arg := range buildCmd {
		cmdstr += " " + arg
	}

	fmt.Printf(`Build working directory: %s`+"\n", sdkpaths.StripRoot(params.WorkDir))

	err := cmd.Exec(cmdstr, &cmd.ExecOpts{
		Stdout: os.Stdout,
		Env:    append(os.Environ(), params.Env...),
		Dir:    params.WorkDir,
	})
	if err != nil {
		return err
	}

	fmt.Println("Module built successfully: " + sdkpaths.StripRoot(filepath.Join(params.WorkDir, outfile)))
	return nil
}

type InstallOpts struct {
	Def       PluginSrcDef
	RemoveSrc bool
	Encrypt   bool
}
