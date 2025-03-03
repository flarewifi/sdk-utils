package tools

import (
	"path/filepath"
	"runtime"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func MainFile() string {
	if runtime.GOOS == "windows" {
		return "main.exe"
	}
	return "main.app"
}

func MainPath() string {
	return filepath.Join(sdkutils.PathAppDir, "main", MainFile())
}
