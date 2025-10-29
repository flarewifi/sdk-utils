//go:build dev

package encdisk

import (
	"log"
	"os"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func (d *EncryptedDisk) Mount() error {
	log.Println("Initializing encrypted disk: ", d.mountpath)
	return sdkutils.FsEmptyDir(d.mountpath)
}

func (d *EncryptedDisk) Unmount() error {
	return os.RemoveAll(d.mountpath)
}
