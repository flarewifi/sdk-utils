// Creates a temporary encrypted directory

package encdisk

import (
	sdkutils "github.com/flarewifi/sdk-utils"
)

type EncryptedDisk struct {
	mountpath string
	file      string
	name      string
	pass      string
}

func NewEncrypedDisk(file string, mountpath string, name string) *EncryptedDisk {
	return &EncryptedDisk{
		mountpath: mountpath,
		file:      file,
		name:      name,
		pass:      sdkutils.RandomStr(16),
	}
}
