package config

import (
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/goccy/go-json"
)

func readConfigFile(f string, out any) error {
	location := filepath.Join(sdkutils.PathConfigDir, f)
	bytes, err := os.ReadFile(location)
	if err != nil {
		// read from defaults
		location = filepath.Join(sdkutils.PathDefaultsDir, f)
		bytes, err = os.ReadFile(location)
		if err != nil {
			return err
		}
	}

	return json.Unmarshal(bytes, out)
}

func writeConfigFile(f string, config any) error {
	bytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	location := filepath.Join(sdkutils.PathConfigDir, f)
	return sdkutils.FsWriteFile(location, bytes)
}
