package main

import (
	"core/utils/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func main() {
	if err := plugins.BuildAssets(sdkutils.PathCoreDir); err != nil {
		panic(err)
	}

	build := &sdkutils.BuildOutput{
		OutputDir: "output/core-files",
		Files: []string{
			"defaults",
			"core/go.mod",
			"core/go.sum",
			"core/sqlc.yml",
			"core/sqlc.postgres.yml",
			"core/sqlc.sqlite.yml",
			"core/package.json",
			"core/package-lock.json",
			"core/plugin.json",
			"core/resources",
			"core/utils",
			"sdk",
			"scripts",
			"plugins/system",
			"go.work.default",
			"start.sh",
		},
		Custom: []sdkutils.BuildOutputCustomEntry{
			{
				Src:  "go.work.default",
				Dest: "go.work",
			},
		},
	}

	if err := build.Run(); err != nil {
		panic(err)
	}
}
