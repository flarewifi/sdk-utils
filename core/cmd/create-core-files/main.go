package main

import "core/build/tools"

func main() {
	build := &tools.BuildOutput{
		OutputDirName: "core-files",
		Files: []string{
			"config/.defaults",
			"core/go.mod",
			"core/plugin.json",
			"core/resources",
			"plugins/system",
			"sdk",
			"go.work.default",
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
