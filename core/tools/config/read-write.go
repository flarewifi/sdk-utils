package config

import (
	"os"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/goccy/go-json"
)

func readConfigFile(f string, out any) error {
	configFile := filepath.Join(sdkutils.PathConfigDir, f)

	b, err := os.ReadFile(configFile)
	if err == nil {
		if err = json.Unmarshal(b, out); err == nil {
			return nil // Success: config read and parsed
		}
	}

	// Fallback: read from defaults on any error (file not found or invalid JSON)
	configFile = filepath.Join(sdkutils.PathDefaultsDir, f)
	b, err = os.ReadFile(configFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, out)
}

func writeConfigFile(f string, config any) error {
	bytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	location := filepath.Join(sdkutils.PathConfigDir, f)
	return sdkutils.FsWriteFile(location, bytes)
}
