package plugins

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	sdkutils "github.com/flarewifi/sdk-utils"
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
	pkg := pluginDir
	if info, infoErr := sdkutils.GetPluginInfoFromPath(pluginDir); infoErr == nil && info.Package != "" {
		pkg = info.Package
	}

	// Tag every failure with the plugin package so a broken asset (e.g. a
	// manifest entry whose file is missing) identifies the offending plugin in
	// the build log instead of just a filesystem path.
	defer func() {
		if err != nil {
			err = fmt.Errorf("plugin %q: %w", pkg, err)
		}
	}()

	fmt.Printf("Building plugin assets for %s in: %s\n", pkg, pluginDir)

	// No `npm install` here: bundling is done by the Go-native esbuild library,
	// which only needs npm to resolve BARE module specifiers (e.g.
	// `require("jquery")`) against node_modules. Every library a plugin uses is
	// vendored into its resources/assets (and shared libs are exposed via the
	// `@flare/lib` esbuild alias → core/resources/assets/lib), so esbuild
	// resolves real file paths with no node/npm on the machine at all. This is
	// what lets on-device plugin recompiles run on apk OpenWRT, whose feeds no
	// longer ship a binary `node`. A bare specifier that wasn't vendored now
	// fails the build with a clear esbuild "could not resolve" error.

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

		// Ensure parent directory exists for nested filenames
		if err = sdkutils.FsEnsureDir(filepath.Dir(indexFile)); err != nil {
			return
		}

		// Import all files into one file
		indexContent := ""
		for _, f := range files {
			assetPath := filepath.Join(pluginDir, AssetsDir, f)
			// A file listed in the manifest but missing on disk is a build error,
			// not a skippable warning: silently dropping it ships a plugin whose
			// bundled JS/CSS is incomplete (broken admin/portal UI) with no failure
			// signal. Block the build so the missing asset is fixed first.
			if !sdkutils.FsExists(assetPath) {
				return results, fmt.Errorf("asset file %q listed in manifest does not exist: %s", f, assetPath)
			}

			rel, err := filepath.Rel(filepath.Dir(indexFile), assetPath)
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

		distPath = filepath.Join(pluginDir, DistDir)
		outfile := filepath.Join(distPath, outname+ext)

		// Ensure parent directory exists for nested filenames
		if err = sdkutils.FsEnsureDir(filepath.Dir(outfile)); err != nil {
			return
		}

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
			// Ensure parent directory exists for nested filenames
			if err = sdkutils.FsEnsureDir(filepath.Dir(out.Path)); err != nil {
				return
			}

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
		}
	}

	return
}
