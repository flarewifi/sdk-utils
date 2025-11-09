package plugins

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	cmd "tools/shell"

	"github.com/evanw/esbuild/pkg/api"
	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	AssetsDir          = "resources/assets"
	DistDir            = "resources/assets/dist"
	AdminManifestJson  = "resources/assets/manifest.admin.json"
	PortalManifestJson = "resources/assets/manifest.portal.json"
	BootManifestJson   = "resources/assets/manifest.boot.json"
)

type Manifest map[string][]string

func BuildAssets(pluginDir string) (err error) {
	fmt.Printf("Building plugin assets in: %s\n", pluginDir)

	if !sdkutils.FsExists(filepath.Join(sdkutils.PathCoreDir, "node_modules")) {
		if _, err := sdkutils.Retry(func() (any, error) {
			if err := cmd.Exec("npm install", &cmd.ExecOpts{Dir: sdkutils.PathCoreDir, Stdout: os.Stdout}); err != nil {
				return nil, fmt.Errorf("failed to install core node modules: %w", err)
			}
			return nil, nil
		}, 3); err != nil {
			return err
		}
	}

	if sdkutils.FsExists(filepath.Join(pluginDir, "package.json")) {
		if _, err := sdkutils.Retry(func() (any, error) {
			err := cmd.Exec("npm install", &cmd.ExecOpts{Dir: pluginDir, Stdout: os.Stdout})
			return nil, err
		}, 3); err != nil {
			return fmt.Errorf("failed to install plugin node modules: %w", err)
		}
	}

	// Clean up dist folder
	if err = os.RemoveAll(filepath.Join(pluginDir, DistDir)); err != nil {
		return
	}

	outManifest := OutputManifest{}

	adminManifestPath := filepath.Join(pluginDir, AdminManifestJson)
	if sdkutils.FsExists(adminManifestPath) {
		var manifest Manifest
		if err = sdkutils.JsonRead(adminManifestPath, &manifest); err != nil {
			return fmt.Errorf("failed to read admin manifest: %w", err)
		}
		fmt.Printf("Compiling assets manifest: %+v\n", manifest)

		if results, err := compileManifest(pluginDir, manifest, api.ES2017); err != nil {
			return fmt.Errorf("failed to compile admin manifest: %w", err)
		} else {
			outManifest.AdminAssets = results
		}
	}

	portalManifestPath := filepath.Join(pluginDir, PortalManifestJson)
	if sdkutils.FsExists(portalManifestPath) {
		var manifest Manifest
		if err = sdkutils.JsonRead(portalManifestPath, &manifest); err != nil {
			return fmt.Errorf("failed to read portal manifest: %w", err)
		}
		fmt.Printf("Compiling assets manifest: %+v\n", manifest)

		if results, err := compileManifest(pluginDir, manifest, api.ES5); err != nil {
			return fmt.Errorf("failed to compile portal manifest: %w", err)
		} else {
			outManifest.PortalAssets = results
		}
	}

	bootManifestPath := filepath.Join(pluginDir, BootManifestJson)
	if sdkutils.FsExists(bootManifestPath) {
		var manifest Manifest
		if err = sdkutils.JsonRead(bootManifestPath, &manifest); err != nil {
			return fmt.Errorf("failed to read boot manifest: %w", err)
		}
		fmt.Printf("Compiling assets manifest: %+v\n", manifest)

		if results, err := compileManifest(pluginDir, manifest, api.ES2017); err != nil {
			return fmt.Errorf("failed to compile boot manifest: %w", err)
		} else {
			outManifest.BootAssets = results
		}
	}

	outManifestFile := filepath.Join(pluginDir, OutManifestJson)
	if err = sdkutils.FsEnsureDir(filepath.Dir(outManifestFile)); err != nil {
		return fmt.Errorf("failed to ensure dist directory: %w", err)
	}

	if err = sdkutils.JsonWrite(outManifestFile, outManifest); err != nil {
		return fmt.Errorf("failed to write output manifest: %w", err)
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

	for filename, files := range manifest {
		if len(files) == 0 {
			continue
		}

		ext := filepath.Ext(filename)
		supportedExts := []string{".js", ".css"}
		if !sdkutils.SliceContains(supportedExts, ext) {
			err = errors.New("Unsupported asset format: " + ext)
			return
		}

		var distPath string
		distPath = filepath.Join(pluginDir, DistDir)
		outname := strings.TrimSuffix(filename, ext)
		indexFile := filepath.Join(distPath, outname+"_index"+ext)

		// Import all files into one file
		indexContent := ""
		for _, f := range files {
			f = filepath.Join(pluginDir, AssetsDir, f)
			if !sdkutils.FsExists(f) {
				fmt.Printf("Warning: file %s does not exist, skipping...\n", f)
				continue
			}

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

		if err = sdkutils.FsWriteFile(indexFile, []byte(indexContent)); err != nil {
			return
		}
		defer os.Remove(indexFile)

		fmt.Printf("Compiling index file: %s: %s\n", indexFile, indexContent)

		distPath = filepath.Join(pluginDir, DistDir)
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
