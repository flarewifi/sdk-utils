//go:build dev

package plugins

import (
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func RandomPluginPath() string {
	return filepath.Join(sdkutils.PathTmpDir, "plugins", sdkutils.RandomStr(16))
}
