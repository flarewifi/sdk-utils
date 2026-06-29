package main

import (
	tools "core/utils"
	toolsenv "core/utils/env"
	"core/utils/translations"
	"flag"
	"fmt"
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func main() {
	outdir := flag.String("outdir", "", "Output directory for mono bin files (required)")
	flag.Parse()

	if *outdir == "" {
		panic("--outdir flag is required")
	}

	outputDir := *outdir
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(sdkutils.PathAppDir, outputDir)
	}

	fmt.Printf("Copying mono bin files to: %s\n", outputDir)

	files := []string{
		"bin/flare",
		"core/go.mod",
		"core/plugin.json",
		"core/sqlc.yml",
		"core/resources/assets/dist",
		"core/resources/assets/public",
		"core/resources/migrations",
		"core/resources/translations",
		"defaults",
		"data/config",
		"plugins/installed",
		"scripts",
	}

	// Per-partner product version stamped by the software-release build. Mono
	// devices read it the same way (core/product.json beside plugin.json). It is
	// MANDATORY: a mono release must carry it so the device reports a product version
	// for update-eligibility, so refuse to package without it. The software-release
	// build stamps it (CloneAndParseRelease); local dev seeds it via
	// gen-product-version, so a missing file here is a real build defect.
	if !sdkutils.FsExists(filepath.Join(sdkutils.PathAppDir, "core/product.json")) {
		panic("core/product.json is missing; refusing to package a mono release without a product version")
	}
	files = append(files, "core/product.json")

	for _, f := range files {
		src := filepath.Join(sdkutils.PathAppDir, f)
		dest := filepath.Join(outputDir, f)
		if err := sdkutils.FsCopy(src, dest); err != nil {
			panic(fmt.Errorf("failed to copy %s to output directory: %w", f, err))
		}
		fmt.Printf("Copied: %s\n", f)
	}

	// Ship the build-appropriate boot script as start.sh (start-mono.sh for mono).
	if err := sdkutils.FsCopy(
		filepath.Join(sdkutils.PathAppDir, tools.StartScriptSrc()),
		filepath.Join(outputDir, "start.sh"),
	); err != nil {
		panic(fmt.Errorf("failed to copy start script to output directory: %w", err))
	}
	fmt.Printf("Copied: %s -> start.sh\n", tools.StartScriptSrc())

	// Minify the JSON translation catalogs in production builds (dev ships the
	// pretty-printed source as-is). Per-language JSON is tiny, so minification
	// replaces the old per-language .tar.gz compression entirely.
	if toolsenv.GO_ENV != toolsenv.ENV_DEV {
		if err := translations.MinifyAllCatalogs(outputDir); err != nil {
			panic(fmt.Errorf("failed to minify translations: %w", err))
		}
	}

	fmt.Println("Mono bin files copy completed successfully.")
}
