//go:build dev

package encdisk

import (
	"os"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func (d *EncryptedDisk) Mount() error {
	return sdkutils.FsEmptyDir(d.mountpath)
}

func (d *EncryptedDisk) Unmount() error {
	return os.RemoveAll(d.mountpath)
}
