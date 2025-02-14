package pkg

import (
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
	CoreAdminGlobalsManifest  = filepath.Join(sdkutils.PathCoreDir, "resources/assets/globals.admin.json")
	CorePortalGlobalsManifest = filepath.Join(sdkutils.PathCoreDir, "resources/assets/globals.portal.json")
	CoreGlobalsBundleManifest = filepath.Join(CoreGlobalsDist, "globals.manifest.json")
)

type GlobalAssetsManifest struct {
	Js  []string `json:"js"`
	Css []string `json:"css"`
}

type GlobalBundleManifest struct {
	AdminJsFile   string `json:"admin_js"`
	AdminCssFile  string `json:"admin_css"`
	PortalJsFile  string `json:"portal_js"`
	PortalCssFile string `json:"portal_css"`
}

func ReadGlobalAssetsManifest() (g GlobalBundleManifest) {
	sdkutils.JsonRead(CoreGlobalsBundleManifest, &g)
	return
}

func BuildGlobalAssets() (err error) {
	if sdkutils.FsExists(CoreGlobalsDist) {
		if err = os.RemoveAll(CoreGlobalsDist); err != nil {
			return
		}
	}

	bundleFile := GlobalBundleManifest{}
	if sdkutils.FsExists(CoreAdminGlobalsManifest) {
		var manifest GlobalAssetsManifest
		var resultFile string
		if err = sdkutils.JsonRead(CoreAdminGlobalsManifest, &manifest); err != nil {
			return
		}

		if resultFile, err = compileGlobalJsAssets(manifest.Js, api.ES2017); err != nil {
			return
		}
		bundleFile.AdminJsFile = resultFile

		if resultFile, err = compileGlobalCssAssets(manifest.Css); err != nil {
			return
		}
		bundleFile.AdminCssFile = resultFile
	}

	if sdkutils.FsExists(CorePortalGlobalsManifest) {
		var manifest GlobalAssetsManifest
		var resultFile string
		if err = sdkutils.JsonRead(CorePortalGlobalsManifest, &manifest); err != nil {
			return
		}

		if resultFile, err = compileGlobalJsAssets(manifest.Js, api.ES5); err != nil {
			return
		}
		bundleFile.PortalJsFile = resultFile

		if resultFile, err = compileGlobalCssAssets(manifest.Css); err != nil {
			return
		}
		bundleFile.PortalCssFile = resultFile
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

	indexFile := filepath.Join(distPath, "globals.index.js")
	indexjs := ""
	for _, js := range jsfiles {
		var relPath string
		jsPath := filepath.Join(CoreAssetsDir, js)
		relPath, err = sdkutils.FsRelativeFromTo(indexFile, jsPath)
		if err != nil {
			return
		}
		indexjs += "require('" + relPath + "');\n"
	}

	if err = os.WriteFile(indexFile, []byte(indexjs), sdkutils.PermFile); err != nil {
		return
	}
	defer os.Remove(indexFile)

	outfile := filepath.Join(distPath, "core-globals.js")
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
	for _, css := range cssFiles {
		var relPath string
		cssPath := filepath.Join(CoreAssetsDir, css)
		relPath, err = sdkutils.FsRelativeFromTo(indexFile, cssPath)
		if err != nil {
			return
		}
		indexCss += "@import '" + relPath + "';\n"
	}

	if err = os.WriteFile(indexFile, []byte(indexCss), sdkutils.PermFile); err != nil {
		return
	}
	defer os.Remove(indexFile)

	outfile := filepath.Join(distPath, "core-globals.css")
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

	return
}
