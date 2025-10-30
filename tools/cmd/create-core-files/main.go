package main

import "tools"

func main() {
	build := &tools.BuildOutput{
		OutputDirName: "core-files",
		Files: []string{
			"defaults",
			"core/go.mod",
			"core/go.sum",
			"core/plugin.json",
			"core/resources",
			"sdk",
			"scripts",
			"tools/go.mod",
			"tools/go.sum",
			"plugins/system",
			"go.work.default",
			"start.sh",
		},
		CustomFiles: []tools.CustomFiles{
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
