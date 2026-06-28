package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"core/utils/plugins"
	sdkapi "sdk/api"

	sdkutils "github.com/flarewifi/sdk-utils"

	"github.com/goccy/go-json"
)

var (
	devkitReleaseDir string
	// devkitFiles are copied verbatim from the build tree into the release. The
	// core is CLOSED SOURCE: NO core Go source ships (core/internal, core/utils,
	// core/cmd, core/sqlc.yml, *.go). What DOES ship: the compiled binaries
	// (bin/flare, bin/livereload, core/plugin.so), non-Go core runtime data
	// (core/resources, *.json), and the module files core/go.{mod,sum} — these are
	// dependency manifests, not logic, and let go.work `use ./core` + fix-workspace
	// resolve normally. The docker scaffolding (docker-compose.yml, Dockerfile,
	// docker-cmd.sh) comes from devkit/extras (devkit-specific variants
	// that need no core source), copied after this list so they win.
	devkitFiles = []string{
		// NOTE: the arch-specific compiled binaries (bin/flare, bin/livereload,
		// core/plugin.so) are intentionally NOT listed here. They ship per
		// architecture — copied to bin/<arch>/… and core/plugin.<arch>.so below —
		// so one devkit runs on both linux/amd64 and linux/arm64 containers.
		// Core module manifests (dependency lists, not logic) — allowed to ship so
		// the workspace resolves ./core normally.
		"core/go.mod",
		"core/go.sum",
		// Core JS deps + manifests. Plugin asset bundling (esbuild) imports shared
		// libs (e.g. alpinejs) from core/node_modules; shipping it prebuilt (~1MB,
		// produced by BuildAssets in the builder) lets plugin asset builds run
		// offline — without it the runtime would `npm install` in core/ and fail
		// when core/package.json is absent. Not logic, just third-party JS.
		"core/package.json",
		"core/package-lock.json",
		"core/node_modules",
		// Core runtime data (no Go logic): the compiled .so reads migrations +
		// translations from core/resources at runtime; the json files carry
		// version metadata (product.json is a prebuilt stand-in so the runtime
		// needs no `go run ./core/cmd/gen-product-version`).
		"core/plugin.json",
		"core/product.json",
		"core/resources",
		// Installed data (plugin.json, resources, migrations) for the statically
		// linked Devkit theme. sysplugin-prepare produced this in the builder; the
		// runtime can't regenerate it (no core source), so LoadSystemPlugins reads
		// the shipped copy.
		"plugins/installed",
		// Public plugin SDK — 3rd-party plugins compile against this. Shipped
		// wholesale, including sdk/mkdocs (the plugin API documentation site) so
		// developers have the API reference locally inside the devkit.
		"sdk",
		// Runtime scaffolding (non-core).
		"defaults",
		"scripts",
		"go.work.default",
		".go-version",
	}
)

func CreateDevkit() {
	// Output is named purely by core version, e.g. flarewifi-devkit-1.2.3.zip.
	devkitReleaseDir = filepath.Join(sdkutils.PathAppDir, "output/devkit", fmt.Sprintf("flarewifi-devkit-%s", plugins.GetCoreInfo().Version))

	// Clean up output path
	if err := sdkutils.FsEmptyDir(filepath.Dir(devkitReleaseDir)); err != nil {
		panic(err)
	}

	// Build the bin/flare cli
	BuildFlareCLI()

	// Build core/plugin.so
	BuildCore(plugins.BuildOpts{})

	// Copy devkit files. data/plugins/system is optional — Flarewifi may ship no
	// system plugins, leaving the directory absent — so append it only when
	// present to avoid panicking on a missing source (see CoreFileSet).
	files := devkitFiles
	if sdkutils.FsExists(sdkutils.PathPluginSystemDir) {
		files = append(files, "data/plugins/system")
	}
	for _, entry := range files {
		srcPath := filepath.Join(sdkutils.PathAppDir, entry)
		destPath := filepath.Join(devkitReleaseDir, entry)
		fmt.Println("Copying: ", sdkutils.StripRootPath(srcPath), " -> ", sdkutils.StripRootPath(destPath))

		if err := sdkutils.FsCopy(srcPath, destPath); err != nil {
			panic(err)
		}
	}

	// Place the compiled, architecture-specific binaries under per-arch paths so a
	// single devkit can run on both linux/amd64 and linux/arm64 containers (Apple
	// Silicon and Windows-on-ARM via WSL2 → arm64; Windows/Linux x86 → amd64; Linux
	// ARM → arm64). A Go -buildmode=plugin + CGO core .so can't be cross-compiled,
	// so each arch is built natively in its own buildx platform pass and
	// merge-devkit.sh unions the two trees. At container boot select-arch.sh copies
	// the set matching `dpkg --print-architecture` into the canonical bin/flare,
	// bin/livereload and core/plugin.so.
	arch := runtime.GOARCH // amd64 | arm64 — matches `dpkg --print-architecture`
	archBinaries := map[string]string{
		"bin/flare":      filepath.Join("bin", arch, "flare"),
		"bin/livereload": filepath.Join("bin", arch, "livereload"),
		"core/plugin.so": fmt.Sprintf("core/plugin.%s.so", arch),
	}
	for src, dst := range archBinaries {
		srcPath := filepath.Join(sdkutils.PathAppDir, src)
		destPath := filepath.Join(devkitReleaseDir, dst)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			panic(err)
		}
		fmt.Println("Copying: ", sdkutils.StripRootPath(srcPath), " -> ", sdkutils.StripRootPath(destPath))
		if err := sdkutils.FsCopy(srcPath, destPath); err != nil {
			panic(err)
		}
	}

	// Copy extra devkit files to the release directory
	extrasPath := filepath.Join(sdkutils.PathAppDir, "devkit/extras")
	fmt.Printf("Copying:  %s -> %s\n", sdkutils.StripRootPath(extrasPath), sdkutils.StripRootPath(devkitReleaseDir))
	err := sdkutils.FsCopyDir(extrasPath, devkitReleaseDir, nil)
	if err != nil {
		panic(err)
	}

	// The Devkit theme system plugin lives under data/plugins/system (committed) —
	// needed at build time to statically link it into core/plugin.so, but it is
	// closed-source Go and must NOT ship as source. Its compiled form lives in
	// core/plugin.so and its RESOURCES ship separately under plugins/installed/.
	// Strip the source tree from the release. (Local-dev-only plugins such as the
	// developer upload/install panel live under data/plugins/devel instead, which
	// is never copied into the release, so they are absent here automatically.)
	// This also leaves data/plugins/system empty at runtime, so the
	// statically-linked plugin is never re-built by `flare build-plugins`.
	sysPluginSrc := filepath.Join(devkitReleaseDir, "data/plugins/system")
	if sdkutils.FsExists(sysPluginSrc) {
		fmt.Println("Removing: ", sdkutils.StripRootPath(sysPluginSrc))
		if err := os.RemoveAll(sysPluginSrc); err != nil {
			panic(err)
		}
	}

	// Generate default application config
	appConfigFile := filepath.Join(devkitReleaseDir, "data/config/application.json")
	appConfig := sdkapi.AppConfig{
		Lang:     "en",
		Currency: "php",
		Secret:   sdkutils.RandomStr(16),
		Channel:  "development",
	}

	b, err := json.MarshalIndent(appConfig, "", "  ")
	if err != nil {
		panic(err)
	}

	if err := os.MkdirAll(filepath.Dir(appConfigFile), 0755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(appConfigFile, b, 0644); err != nil {
		panic(err)
	}

	fmt.Println("Application config created: ", sdkutils.StripRootPath(appConfigFile))

	// Make the Devkit theme the DEFAULT admin + portal theme. Without this the
	// theme config defaults to the built-in core theme, which renders the bare
	// fallback layout (and a "select a valid theme" warning) — the devkit should
	// boot straight into the Devkit theme so a developer's plugin UI has a real
	// host to render in. com.flarego.devkit ships under plugins/installed, so
	// isThemeValid() accepts it.
	themesConfigFile := filepath.Join(devkitReleaseDir, "data/config/themes.json")
	themesConfig := map[string]string{
		"admin":  "com.flarego.devkit",
		"portal": "com.flarego.devkit",
	}
	tb, err := json.MarshalIndent(themesConfig, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(themesConfigFile, tb, 0644); err != nil {
		panic(err)
	}

	fmt.Println("Default theme config created (Devkit theme): ", sdkutils.StripRootPath(themesConfigFile))

	// Clean up core template files
	var templateFiles []string
	if err := sdkutils.FsListFiles(filepath.Join(devkitReleaseDir, "core/resources/views"), &templateFiles, true); err != nil {
		panic(err)
	}

	for _, file := range templateFiles {
		if filepath.Ext(file) == ".templ" || strings.HasSuffix(file, "_templ.go") {
			fmt.Println("Removing: ", sdkutils.StripRootPath(file))
			if err := os.Remove(file); err != nil {
				panic(err)
			}
		}
	}

	// Prune empty directories so they never ship in the release. Stripping the
	// closed-source core's templates above hollows out view dirs that held only
	// .templ/_templ.go files (boot, activation, admin/power, the fallback theme
	// tree, etc.); the system-plugin source removal can do the same. Loop
	// because each pass only removes the current deepest empties — collapsing them
	// exposes newly-empty parents (e.g. themes/fallback/admin nests 3 deep). Dirs
	// that must exist empty at runtime are kept by their committed .keep files
	// (core/, main/, data/plugins/{local,devel}/), so they are never truly empty.
	for {
		var emptyDirs []string
		if err := sdkutils.FsFindEmptyDirs(devkitReleaseDir, &emptyDirs); err != nil {
			panic(err)
		}
		if len(emptyDirs) == 0 {
			break
		}
		for _, dir := range emptyDirs {
			fmt.Println("Removing empty dir: ", sdkutils.StripRootPath(dir))
			if err := os.Remove(dir); err != nil {
				panic(err)
			}
		}
	}

	// Multi-arch builds skip zipping here: each buildx platform pass emits only its
	// own arch tree, and merge-devkit.sh unions the per-arch trees into one fat zip
	// host-side. A plain single-arch `create-devkit` still zips for local use.
	if os.Getenv("DEVKIT_NO_ZIP") != "" {
		fmt.Println("Devkit tree created (DEVKIT_NO_ZIP set, skipping zip): ", sdkutils.StripRootPath(devkitReleaseDir))
		return
	}

	// Compress devkit release files
	file := filepath.Base(devkitReleaseDir) + ".zip"
	dir := filepath.Dir(devkitReleaseDir)
	zipPath := filepath.Join(dir, file)
	if err := sdkutils.CompressZip(devkitReleaseDir, zipPath); err != nil {
		panic(err)
	}

	fmt.Println("Devkit created: ", sdkutils.StripRootPath(zipPath))
}
