package boot

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	cmd "tools/shell"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// Install ipk files in ./packages directory
func InitOpkg() {
	fmt.Println("Installing ipk packages...")

	var files []string

	packagesDir := filepath.Join(sdkutils.PathAppDir, "packages")
	if err := sdkutils.FsListFiles(packagesDir, &files, true); err != nil {
		log.Printf("Error listing files in packages in %s: %v", packagesDir, err.Error())
		return
	}

	for _, f := range files {
		if filepath.Ext(f) == ".ipk" {
			if err := cmd.Exec("opkg install "+f, &cmd.ExecOpts{
				Stdout: os.Stdout,
				Stderr: os.Stderr,
			}); err != nil {
				log.Printf("Error installing ipk file %s: %v", f, err.Error())
				return
			}

			// remove file if installed successfully
			os.Remove(f)
		}
	}
}
