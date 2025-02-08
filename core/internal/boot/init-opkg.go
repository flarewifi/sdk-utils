package boot

import (
	"core/internal/api"
	"core/internal/utils/cmd"
	"fmt"
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func InitOpkg(bp *api.BootProgress) {
	var files []string

	packagesDir := filepath.Join(sdkutils.PathAppDir, "packages")
	if err := sdkutils.FsListFiles(packagesDir, &files, true); err != nil {
		bp.AppendLog(fmt.Sprintf("Error listing files in packages in %s: %v", packagesDir, err.Error()))
		return
	}

	for _, f := range files {
		if filepath.Ext(f) == ".ipk" {
			bp.AppendLog("Installing ipk file: " + f)

			if err := cmd.Exec("opkg install "+f, &cmd.ExecOpts{
				Stdout: os.Stdout,
				Stderr: os.Stderr,
			}); err != nil {
				bp.AppendLog(fmt.Sprintf("Error installing ipk file %s: %v", f, err.Error()))
				return
			}

			// remove file if installed successfully
			os.Remove(f)
		}
	}
}
