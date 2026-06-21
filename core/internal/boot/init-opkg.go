package boot

import (
	cmd "core/utils/shell"
	"os"
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// Install ipk files in ./packages directory
func InitOpkg() {
	var files []string

	packagesDir := filepath.Join(sdkutils.PathAppDir, "packages")
	if err := sdkutils.FsListFiles(packagesDir, &files, true); err != nil {
		return
	}

	for _, f := range files {
		if filepath.Ext(f) == ".ipk" {
			if err := cmd.Exec("opkg install "+f, &cmd.ExecOpts{
				Stdout: os.Stdout,
				Stderr: os.Stderr,
			}); err != nil {
				return
			}

			// remove file if installed successfully
			os.Remove(f)
		}
	}
}
