//go:build dev

package machineuid

import (
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func GetMachineUID() string {
	f := filepath.Join(sdkutils.PathAppDir, "MACHINE_UID")
	uid, err := sdkutils.FsReadFile(f)
	if err != nil {
		return "machine_001"
	}

	return uid
}
