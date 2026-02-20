//go:build dev

package machineuid

import (
	"path/filepath"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

// GetMachineUID returns a unique identifier for the dev environment.
// In dev mode, machine ID never changes, so always returns ("", machineID)
func GetMachineUID() (string, string) {
	f := filepath.Join(sdkutils.PathAppDir, "MACHINE_UID")
	uid, err := sdkutils.FsReadFile(f)
	if err != nil {
		return "", "machine_001"
	}

	return "", strings.TrimSpace(uid)
}
