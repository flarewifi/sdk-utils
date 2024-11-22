package config

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-json"

	sdkpaths "github.com/flarehotspot/go-utils/paths"
)

func readConfigFile(f string, out interface{}) error {
	location := filepath.Join(sdkpaths.ConfigDir, f)
	bytes, err := os.ReadFile(location)
	if err != nil {
		// read from defaults
		location = filepath.Join(sdkpaths.ConfigDir, ".defaults", f)
		bytes, err = os.ReadFile(location)
		if err != nil {
			return err
		}
	}

	return json.Unmarshal(bytes, out)
}

func writeConfigFile(f string, config interface{}) error {
	bytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	location := filepath.Join(sdkpaths.ConfigDir, f)
	return os.WriteFile(location, bytes, 0644)
}
