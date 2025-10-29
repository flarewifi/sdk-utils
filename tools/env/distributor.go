package env

import (
	"os"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func GetDistributorCode() (string, error) {
	var p string
	if GO_ENV != ENV_PRODUCTION {
		p = filepath.Join(sdkutils.PathAppDir, ".files", "distributor")
	} else {
		p = "/etc/distributor"
	}

	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(b)), nil
}
