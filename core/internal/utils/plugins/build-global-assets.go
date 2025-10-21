package plugins

import (
	"core/internal/utils/cmd"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	CoreAssetsDir             = filepath.Join(sdkutils.PathCoreDir, "resources/assets")
	CoreGlobalsDist           = filepath.Join(CoreAssetsDir, "dist/globals")
	CoreBootGlobalsManifest   = filepath.Join(sdkutils.PathCoreDir, "resources/assets/manifest.boot.json")
	CoreGlobalsBundleManifest = filepath.Join(CoreGlobalsDist, "globals.manifest.json")
	GlobalJsIndex             = "global.js"
	GlobalCssIndex            = "global.css"
)

type GlobalAssetsManifest struct {
	Js  []string `json:"js"`
	Css []string `json:"css"`
}

type GlobalBundleManifest struct {
	AdminJsFile    string `json:"admin_js"`
	AdminCssFile   string `json:"admin_css"`
	PortalJsFile   string `json:"portal_js"`
	PortalCssFile  string `json:"portal_css"`
	BootingJsFile  string `json:"booting_js"`
	BootingCssFile string `json:"booting_css"`
}

type PluginGlobalAssets struct {
	AdminJsFiles   []string
	AdminCssFiles  []string
	PortalJsFiles  []string
	PortalCssFiles []string
}

type PluginManifest struct {
	GlobalJs  []string `json:"global.js"`
	GlobalCss []string `json:"global.css"`
}

func GetGlobalAssets(pluginDirs []string) (globalAssets PluginGlobalAssets, err error) {
	for _, pluginDir := range pluginDirs {
		adminManifestFile := filepath.Join(pluginDir, "resources/assets/manifest.admin.json")
		portalManifestFile := filepath.Join(pluginDir, "resources/assets/manifest.portal.json")

		var adminManifest, portalManifest PluginManifest

		if sdkutils.FsExists(adminManifestFile) {
			if err = sdkutils.JsonRead(adminManifestFile, &adminManifest); err != nil {
				return
			}
			for _, f := range adminManifest.GlobalJs {
				file := filepath.Join(pluginDir, "resources/assets", f)
				globalAssets.AdminJsFiles = append(globalAssets.AdminJsFiles, file)
			}
			for _, f := range adminManifest.GlobalCss {
				file := filepath.Join(pluginDir, "resources/assets", f)
				globalAssets.AdminCssFiles = append(globalAssets.AdminCssFiles, file)
			}
		} else {
			fmt.Printf("Admin manifest file not found: %s\n", adminManifestFile)
		}

		if sdkutils.FsExists(portalManifestFile) {
			if err = sdkutils.JsonRead(portalManifestFile, &portalManifest); err != nil {
				return
			}
			fmt.Println("Portal manifest:", portalManifest)
			for _, f := range portalManifest.GlobalJs {
				file := filepath.Join(pluginDir, "resources/assets", f)
				globalAssets.PortalJsFiles = append(globalAssets.PortalJsFiles, file)
			}
			for _, f := range portalManifest.GlobalCss {
				file := filepath.Join(pluginDir, "resources/assets", f)
				globalAssets.PortalCssFiles = append(globalAssets.PortalCssFiles, file)
			}
		} else {
			fmt.Printf("Portal manifest file not found: %s\n", portalManifestFile)
		}
	}

	fmt.Printf("Total plugin admin js files: %v\n", globalAssets.AdminJsFiles)
	fmt.Printf("Total plugin admin css files: %v\n", globalAssets.AdminCssFiles)
	return
}

func ReadGlobalAssetsManifest() (g GlobalBundleManifest) {
	sdkutils.JsonRead(CoreGlobalsBundleManifest, &g)
	return
}

func BuildGlobalAssets(pluginDirs []string) (err error) {
	if _, err := sdkutils.Retry(func() (any, error) {
		err := cmd.Exec("npm install", &cmd.ExecOpts{Dir: sdkutils.PathCoreDir})
		return nil, err
	}, 3); err != nil {
		return err
	}

	if sdkutils.FsExists(CoreGlobalsDist) {
		if err = os.RemoveAll(CoreGlobalsDist); err != nil {
			return err
		}
	}

	globalAssets, err := GetGlobalAssets(pluginDirs)
	if err != nil {
		return err
	}

	bundleFile := GlobalBundleManifest{}

	adminJsResultFile, err := compileGlobalJsAssets(globalAssets.AdminJsFiles, api.ES2017)
	if err != nil {
		return err
	}

	adminCssResultFile, err := compileGlobalCssAssets(globalAssets.AdminCssFiles)
	if err != nil {
		return err
	}

	bundleFile.AdminJsFile = adminJsResultFile
	bundleFile.AdminCssFile = adminCssResultFile

	portalJsResultFile, err := compileGlobalJsAssets(globalAssets.PortalJsFiles, api.ES5)
	if err != nil {
		return err
	}

	portalCssResultFile, err := compileGlobalCssAssets(globalAssets.PortalCssFiles)
	if err != nil {
		return err
	}
	bundleFile.PortalJsFile = portalJsResultFile
	bundleFile.PortalCssFile = portalCssResultFile

	if sdkutils.FsExists(CoreBootGlobalsManifest) {
		var manifest GlobalAssetsManifest
		var resultFile string
		if err = sdkutils.JsonRead(CoreBootGlobalsManifest, &manifest); err != nil {
			return
		}

		var bootJsFiles, bootCssFiles []string

		for _, js := range manifest.Js {
			bootJsFiles = append(bootJsFiles, filepath.Join(CoreAssetsDir, js))
		}

		for _, css := range manifest.Css {
			bootCssFiles = append(bootCssFiles, filepath.Join(CoreAssetsDir, css))
		}

		if resultFile, err = compileGlobalJsAssets(bootJsFiles, api.ES2017); err != nil {
			return
		}
		bundleFile.BootingJsFile = resultFile

		if resultFile, err = compileGlobalCssAssets(bootCssFiles); err != nil {
			return
		}
		bundleFile.BootingCssFile = resultFile
	}

	if err = sdkutils.JsonWrite(CoreGlobalsBundleManifest, bundleFile); err != nil {
		return
	}

	return
}

func compileGlobalJsAssets(jsfiles []string, target api.Target) (resultFile string, err error) {
	distPath := filepath.Join(CoreGlobalsDist, "js")
	if err = sdkutils.FsEnsureDir(distPath); err != nil {
		return
	}

	indexFile := filepath.Join(distPath, "globals-index.js")
	indexjs := ""
	for _, jsPath := range jsfiles {
		var relPath string
		relPath, err = filepath.Rel(filepath.Dir(indexFile), jsPath)
		if err != nil {
			return
		}
		indexjs += "require('" + relPath + "');\n"
	}

	if err = os.WriteFile(indexFile, []byte(indexjs), sdkutils.PermFile); err != nil {
		return
	}
	defer os.Remove(indexFile)

	outfile := filepath.Join(distPath, "globals-compiled.js")
	result := EsbuildJs(indexFile, outfile, target)

	if len(result.Errors) > 0 {
		err = fmt.Errorf("%s: %v", outfile, result.Errors)
		return
	}

	if len(result.Warnings) > 0 {
		err = fmt.Errorf("Warnings: %v", result.Warnings)
		return
	}

	for _, outfile := range result.OutputFiles {
		f := filepath.Base(outfile.Path)
		outpath := filepath.Join(distPath, f)
		if err = sdkutils.FsEnsureDir(filepath.Dir(outpath)); err != nil {
			return
		}
		if err = os.WriteFile(outpath, outfile.Contents, sdkutils.PermFile); err != nil {
			return
		}
		fmt.Printf("Outputfile written to: %s\n", outpath)

		if filepath.Ext(f) == ".js" {
			resultFile = path.Join("globals", "js", f)
		}
	}

	return
}

func compileGlobalCssAssets(cssFiles []string) (resultFile string, err error) {
	distPath := filepath.Join(CoreGlobalsDist, "css")
	if err = sdkutils.FsEnsureDir(distPath); err != nil {
		return
	}

	indexFile := filepath.Join(distPath, "globals-index.css")
	indexCss := ""
	for _, cssPath := range cssFiles {
		var relPath string
		relPath, err = filepath.Rel(filepath.Dir(indexFile), cssPath)
		if err != nil {
			return
		}
		indexCss += "@import '" + relPath + "';\n"
	}

	if err = os.WriteFile(indexFile, []byte(indexCss), sdkutils.PermFile); err != nil {
		return
	}
	defer os.Remove(indexFile)

	outfile := filepath.Join(distPath, "globals-compiled.css")
	result := EsbuildCss(indexFile, outfile)

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			err = fmt.Errorf("failed to compile global css file: %v", e)
		}
		return
	}

	if len(result.Warnings) > 0 {
		err = fmt.Errorf("Warnings: %v", result.Warnings)
		return
	}

	for _, outfile := range result.OutputFiles {
		f := filepath.Base(outfile.Path)
		outpath := filepath.Join(distPath, f)
		if err = sdkutils.FsEnsureDir(filepath.Dir(outpath)); err != nil {
			return
		}
		if err = os.WriteFile(outpath, outfile.Contents, sdkutils.PermFile); err != nil {
			return
		}
		fmt.Printf("Outputfile written to: %s\n", outpath)

		if filepath.Ext(f) == ".css" {
			resultFile = path.Join("globals", "css", f)
		}
	}

	fmt.Println("Compiled global assets successfully.\n\n")

	return
}
