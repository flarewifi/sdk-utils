package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
)

var (
	CoreAssetsDir             = filepath.Join(sdkpaths.CoreDir, "resources/assets")
	CoreGlobalsDist           = filepath.Join(CoreAssetsDir, "dist/globals")
	CoreAdminGlobalsManifest  = filepath.Join(sdkpaths.CoreDir, "resources/assets/globals.admin.json")
	CorePortalGlobalsManifest = filepath.Join(sdkpaths.CoreDir, "resources/assets/globals.portal.json")
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
	sdkfs.ReadJson(CoreGlobalsBundleManifest, &g)
	return
}

func BuildGlobalAssets() (err error) {
	if sdkfs.Exists(CoreGlobalsDist) {
		if err = os.RemoveAll(CoreGlobalsDist); err != nil {
			return
		}
	}

	bundleFile := GlobalBundleManifest{}
	if sdkfs.Exists(CoreAdminGlobalsManifest) {
		var manifest GlobalAssetsManifest
		var resultFile string
		if err = sdkfs.ReadJson(CoreAdminGlobalsManifest, &manifest); err != nil {
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

	if sdkfs.Exists(CorePortalGlobalsManifest) {
		var manifest GlobalAssetsManifest
		var resultFile string
		if err = sdkfs.ReadJson(CorePortalGlobalsManifest, &manifest); err != nil {
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

	if err = sdkfs.WriteJson(CoreGlobalsBundleManifest, bundleFile); err != nil {
		return
	}

	return
}

func compileGlobalJsAssets(jsfiles []string, target api.Target) (resultFile string, err error) {
	distPath := filepath.Join(CoreGlobalsDist, "js")
	if err = sdkfs.EnsureDir(distPath); err != nil {
		return
	}

	indexFile := filepath.Join(distPath, "globals.index.js")
	indexjs := ""
	for _, js := range jsfiles {
		var relPath string
		jsPath := filepath.Join(CoreAssetsDir, js)
		relPath, err = sdkpaths.RelativeFromTo(indexFile, jsPath)
		if err != nil {
			return
		}
		indexjs += "require('" + relPath + "');\n"
	}

	if err = os.WriteFile(indexFile, []byte(indexjs), sdkfs.PermFile); err != nil {
		return
	}
	defer os.Remove(indexFile)

	outfile := filepath.Join(distPath, "globals.js")
	result := EsbuildJs(indexFile, outfile, target)

	if len(result.Errors) > 0 {
		err = fmt.Errorf("failed to compile global js file: %v", result.Errors)
		return
	}

	if len(result.Warnings) > 0 {
		err = fmt.Errorf("Warnings: %v", result.Warnings)
		return
	}

	for _, outfile := range result.OutputFiles {
		f := filepath.Base(outfile.Path)
		outpath := filepath.Join(distPath, f)
		if err = sdkfs.EnsureDir(filepath.Dir(outpath)); err != nil {
			return
		}
		if err = os.WriteFile(outpath, outfile.Contents, sdkfs.PermFile); err != nil {
			return
		}
		fmt.Printf("Outputfile written to: %s\n", outpath)

		if filepath.Ext(f) == ".js" {
			resultFile = filepath.Join("globals/js", f)
		}
	}

	return
}

func compileGlobalCssAssets(cssFiles []string) (resultFile string, err error) {
	distPath := filepath.Join(CoreGlobalsDist, "css")
	if err = sdkfs.EnsureDir(distPath); err != nil {
		return
	}

	indexFile := filepath.Join(distPath, "globals-index.css")
	indexCss := ""
	for _, css := range cssFiles {
		var relPath string
		cssPath := filepath.Join(CoreAssetsDir, css)
		relPath, err = sdkpaths.RelativeFromTo(indexFile, cssPath)
		if err != nil {
			return
		}
		indexCss += "import '" + relPath + "';\n"
	}

	if err = os.WriteFile(indexFile, []byte(indexCss), sdkfs.PermFile); err != nil {
		return
	}
	defer os.Remove(indexFile)

	outfile := filepath.Join(distPath, "globals.css")
	result := EsbuildCss(indexFile, outfile)

	if len(result.Errors) > 0 {
		err = fmt.Errorf("failed to compile global css file: %v", result.Errors)
		return
	}

	if len(result.Warnings) > 0 {
		err = fmt.Errorf("Warnings: %v", result.Warnings)
		return
	}

	for _, outfile := range result.OutputFiles {
		f := filepath.Base(outfile.Path)
		outpath := filepath.Join(distPath, f)
		if err = sdkfs.EnsureDir(filepath.Dir(outpath)); err != nil {
			return
		}
		if err = os.WriteFile(outpath, outfile.Contents, sdkfs.PermFile); err != nil {
			return
		}
		fmt.Printf("Outputfile written to: %s\n", outpath)

		if filepath.Ext(f) == ".css" {
			resultFile = filepath.Join("globals/css", f)
		}
	}

	return
}
