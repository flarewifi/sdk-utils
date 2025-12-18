package main

import (
	"flag"
	"fmt"
	"path/filepath"
	toolsenv "core/tools/env"

	sdkutils "github.com/flarehotspot/sdk-utils"
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
		"core/sqlc.postgres.yml",
		"core/sqlc.sqlite.yml",
		"core/resources/assets/dist",
		"core/resources/assets/public",
		"core/resources/migrations",
		"core/resources/translations",
		"defaults",
		"data/config",
		"plugins/installed",
		"scripts",
		"start.sh",
	}

	for _, f := range files {
		src := filepath.Join(sdkutils.PathAppDir, f)
		dest := filepath.Join(outputDir, f)
		if err := sdkutils.FsCopy(src, dest); err != nil {
			panic(fmt.Errorf("failed to copy %s to output directory: %w", f, err))
		}
		fmt.Printf("Copied: %s\n", f)
	}

	// Skip translation compression in dev mode
	if toolsenv.GO_ENV != toolsenv.ENV_DEV {
		if err := sdkutils.CompressAllTranslations(outputDir); err != nil {
			panic(fmt.Errorf("failed to compress translations: %w", err))
		}
	}

	fmt.Println("Mono bin files copy completed successfully.")
}
