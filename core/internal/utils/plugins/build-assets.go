package plugins

import (
	"core/internal/utils/cmd"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	sdkutils "github.com/flarehotspot/sdk-utils"
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
	if err := sdkutils.JsonRead(manifestFile, &manifest); err != nil {
		return emptyManifest
	}

	return manifest
}

func BuildAssets(pluginDir string) (err error) {

	if sdkutils.FsExists(filepath.Join(pluginDir, "package.json")) {
		if _, err := sdkutils.Retry(func() (any, error) {
			err := cmd.Exec("npm install", &cmd.ExecOpts{Dir: pluginDir, Stdout: os.Stdout})
			return nil, err
		}, 3); err != nil {
			return err
		}
		defer cmd.Exec("npm cache clean --force", &cmd.ExecOpts{})
	}
	defer os.RemoveAll(filepath.Join(pluginDir, "node_modules"))

	// Clean up dist folder
	distPath, err := getDistPath(pluginDir)
	if err != nil {
		return err
	}
	if err = os.RemoveAll(distPath); err != nil {
		return
	}

	outManifest := OutputManifest{}

	adminManifestPath := filepath.Join(pluginDir, AdminManifestJson)
	if sdkutils.FsExists(adminManifestPath) {
		var manifest Manifest
		if err = sdkutils.JsonRead(adminManifestPath, &manifest); err != nil {
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
	if sdkutils.FsExists(portalManifestPath) {
		var manifest Manifest
		if err = sdkutils.JsonRead(portalManifestPath, &manifest); err != nil {
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
	if err = sdkutils.FsEnsureDir(filepath.Dir(outManifestFile)); err != nil {
		return err
	}

	if err = sdkutils.JsonWrite(outManifestFile, outManifest); err != nil {
		return err
	}

	return nil
}

func compileManifest(pluginDir string, manifest Manifest, target api.Target) (results CompileResults, err error) {
	results = CompileResults{
		Scripts: make(map[string]string),
		Styles:  make(map[string]string),
	}

	pluginAbsPath, err := filepath.Abs(pluginDir)
	if err != nil {
		return
	}

	// Gather global files
	var globalScrips, globalStyles []string
	for k, files := range manifest {
		switch k {
		case "globals.js":
			globalScrips = append(globalScrips, files...)
		case "globals.css":
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
		if !sdkutils.SliceContains(supportedExts, ext) {
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

		var distPath string
		distPath, err = getDistPath(pluginDir)
		if err != nil {
			return
		}
		outname := strings.TrimSuffix(filename, ext)
		indexFile := filepath.Join(distPath, outname+"_index"+ext)

		// Import all files into one file
		indexContent := ""
		for _, f := range files {
			f = filepath.Join(pluginDir, AssetsDir, f)
			rel, err := filepath.Rel(filepath.Dir(indexFile), f)
			if err != nil {
				return results, err
			}

			switch ext {
			case ".js":
				indexContent += fmt.Sprintf("require('%s');\n", rel)
			case ".css":
				indexContent += fmt.Sprintf("@import '%s';\n", rel)
			}
		}

		if err = sdkutils.FsEnsureDir(filepath.Dir(indexFile)); err != nil {
			return
		}
		if err = os.WriteFile(indexFile, []byte(indexContent), sdkutils.PermFile); err != nil {
			return
		}
		defer os.Remove(indexFile)

		fmt.Printf("Compiling index file: %s: %s\n", indexFile, indexContent)

		distPath, err = getDistPath(pluginDir)
		if err != nil {
			return
		}
		outfile := filepath.Join(distPath, outname+ext)

		var result api.BuildResult
		switch ext {
		case ".js":
			result = EsbuildJs(indexFile, outfile, target)
		case ".css":
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
			fmt.Println("OutputFile: ", out.Path)

			if err = sdkutils.FsWriteFile(out.Path, out.Contents); err != nil {
				return
			}

			if filepath.Ext(out.Path) == ext {
				distPathPrefix := filepath.Join(pluginAbsPath, AssetsDir, "dist")
				fileIndex := strings.TrimPrefix(out.Path, distPathPrefix)

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

func getDistPath(pluginDir string) (string, error) {
	info, err := sdkutils.GetPluginInfoFromPath(pluginDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(pluginDir, DistDir, info.Package), nil
}
