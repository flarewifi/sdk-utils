package main

import "core/build/tools"

func main() {
	build := &tools.BuildOutput{
		OutputDirName: "core-files",
		Files: []string{
			"core/go.mod",
			"core/plugin.json",
			"core/resources",
			"sdk",
			"go.work.default",
			"plugins/system",
			"data/config/.defaults",
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
