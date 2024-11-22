package main

import "core/build/tools"

func main() {
	build := &tools.BuildOutput{
		OutputDirName: "core-files",
		Files: []string{
			"config/.defaults",
			"core/go.mod",
			"core/plugin.json",
			"core/support.json",
			"core/resources",
			"plugins/system",
			"sdk",
			// "utils",
			"go.work.default",
		},
		ExtraFiles: []tools.ExtraFiles{
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
