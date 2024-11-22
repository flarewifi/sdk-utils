package pkg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	sdkfs "github.com/flarehotspot/go-utils/fs"
	sdkpaths "github.com/flarehotspot/go-utils/paths"
	sdkslices "github.com/flarehotspot/go-utils/slices"
)

const (
	AssetsDir          = "resources/assets"
	DistDir            = "resources/assets/dist"
	AdminManifestJson  = "resources/assets/manifest.admin.json"
	PortalManifestJson = "resources/assets/manifest.portal.json"
	OutManifestJson    = "resources/assets/dist/manifest.json"
)

type Manifest map[string][]string

type CompileResults struct {
	Scripts map[string]string
	Styles  map[string]string
}

type OutputManifest struct {
	AdminAssets  CompileResults `json:"admin"`
	PortalAssets CompileResults `json:"portal"`
}

func GetAssetManifest(pluginDir string) OutputManifest {
	manifestFile := filepath.Join(pluginDir, OutManifestJson)
	emptyRes := CompileResults{
		Scripts: make(map[string]string),
		Styles:  make(map[string]string),
	}
	emptyManifest := OutputManifest{
		AdminAssets:  emptyRes,
		PortalAssets: emptyRes,
	}

	var manifest OutputManifest
	if err := sdkfs.ReadJson(manifestFile, &manifest); err != nil {
		return emptyManifest
	}

	return manifest
}

func BuildAssets(pluginDir string) (err error) {
	// Clean up dist folder
	distPath := filepath.Join(pluginDir, "resources/assets/dist")
	if err = os.RemoveAll(distPath); err != nil {
		return
	}

	if err = LinkNodeModulesLib(pluginDir); err != nil {
		return
	}

	defer os.RemoveAll(filepath.Join(pluginDir, "node_modules"))

	outManifest := OutputManifest{}

	adminManifestPath := filepath.Join(pluginDir, AdminManifestJson)
	if sdkfs.Exists(adminManifestPath) {
		var manifest Manifest
		if err = sdkfs.ReadJson(adminManifestPath, &manifest); err != nil {
			return err
		}
		fmt.Printf("Compiling assets manifest: %+v\n", manifest)

		if results, err := compileManifest(pluginDir, manifest, api.ES2017); err != nil {
			return err
		} else {
			outManifest.AdminAssets = results
		}
	}

	portalManifestPath := filepath.Join(pluginDir, PortalManifestJson)
	if sdkfs.Exists(portalManifestPath) {
		var manifest Manifest
		if err = sdkfs.ReadJson(portalManifestPath, &manifest); err != nil {
			return err
		}
		fmt.Printf("Compiling assets manifest: %+v\n", manifest)

		if results, err := compileManifest(pluginDir, manifest, api.ES5); err != nil {
			return err
		} else {
			outManifest.PortalAssets = results
		}
	}

	outManifestFile := filepath.Join(pluginDir, OutManifestJson)
	if err = sdkfs.EnsureDir(filepath.Dir(outManifestFile)); err != nil {
		return err
	}

	if err = sdkfs.WriteJson(outManifestFile, outManifest); err != nil {
		return err
	}

	return nil
}

func compileManifest(pluginDir string, manifest Manifest, target api.Target) (results CompileResults, err error) {
	results = CompileResults{
		Scripts: make(map[string]string),
		Styles:  make(map[string]string),
	}

	// Gather global files
	var globalScrips, globalStyles []string
	for k, files := range manifest {
		if k == "globals.js" {
			globalScrips = append(globalScrips, files...)
		} else if k == "globals.css" {
			globalStyles = append(globalStyles, files...)
		}
	}

	for filename, files := range manifest {
		// Don't output global scripts and styles, they are already bundled in non-global files
		if filename == "globals.js" || filename == "globals.css" {
			continue
		}

		// TODO: check if scripts is directory and loadd all files inside it
		ext := filepath.Ext(filename)
		supportedExts := []string{".js", ".css"}
		if !sdkslices.Contains(supportedExts, ext) {
			err = errors.New("Unsupported asset format: " + ext)
			return
		}

		var globalFiles []string
		if ext == ".js" {
			globalFiles = globalScrips
		} else {
			globalFiles = globalStyles
		}

		// Bundle global files with files
		files = append(globalFiles, files...)

		distPath := filepath.Join(pluginDir, AssetsDir, "dist", strings.TrimPrefix(ext, "."))
		outname := strings.TrimSuffix(filename, ext)
		indexFile := filepath.Join(distPath, outname+"_index"+ext)

		// Import all files into one file
		indexContent := ""
		for _, f := range files {
			f = filepath.Join(pluginDir, AssetsDir, f)
			rel, err := sdkpaths.RelativeFromTo(indexFile, f)
			if err != nil {
				return results, err
			}

			if ext == ".js" {
				indexContent += fmt.Sprintf("require('%s');\n", rel)
			} else if ext == ".css" {
				indexContent += fmt.Sprintf("@import '%s';\n", rel)
			}
		}

		if err = sdkfs.EnsureDir(filepath.Dir(indexFile)); err != nil {
			return
		}
		if err = os.WriteFile(indexFile, []byte(indexContent), sdkfs.PermFile); err != nil {
			return
		}
		defer os.Remove(indexFile)

		fmt.Printf("Compiling index file: %s: %s\n", indexFile, indexContent)

		outfile := filepath.Join(distPath, outname+ext)

		var result api.BuildResult
		if ext == ".js" {
			result = EsbuildJs(indexFile, outfile, target)
		} else if ext == ".css" {
			result = EsbuildCss(indexFile, outfile)
		}

		if len(result.Errors) > 0 {
			err = fmt.Errorf("failed to compile %s %v", ext, result.Errors)
			return
		}

		if len(result.Warnings) > 0 {
			err = fmt.Errorf("%s warnings: %v", ext, result.Warnings)
			return
		}

		for _, out := range result.OutputFiles {
			f := filepath.Base(out.Path)
			outpath := filepath.Join(distPath, f)
			if err = sdkfs.EnsureDir(filepath.Dir(outpath)); err != nil {
				return
			}
			if err = os.WriteFile(outpath, out.Contents, sdkfs.PermFile); err != nil {
				return
			}
			if filepath.Ext(out.Path) == ext {
				fileIndex := filepath.Join(strings.TrimPrefix(ext, "."), f)

				switch filepath.Ext(out.Path) {
				case ".js":
					results.Scripts[filename] = fileIndex
				case ".css":
					results.Styles[filename] = fileIndex
				}
			}
			fmt.Printf("Outputfile written to: %s\n", out.Path)
		}
	}

	return
}
