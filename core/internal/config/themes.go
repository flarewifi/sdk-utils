package config

import (
	"path/filepath"
	"sync"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	themesConfigJsonFile = "themes.json"
	defaultThemePlugin   = "com.flarego.default-theme"
)

var (
	themeCfgMu    = sync.RWMutex{}
	themeCfgCache *ThemesConfig
)

type ThemesConfig struct {
	PortalThemePkg string `json:"portal"`
	AdminThemePkg  string `json:"admin"`
}

func ReadThemesConfig() (ThemesConfig, error) {
	themeCfgMu.RLock()
	if themeCfgCache != nil {
		defer themeCfgMu.RUnlock()
		// prevent cache modification
		return ThemesConfig{
			PortalThemePkg: themeCfgCache.PortalThemePkg,
			AdminThemePkg:  themeCfgCache.AdminThemePkg,
		}, nil
	}
	themeCfgMu.RUnlock()

	var cfg ThemesConfig
	if err := readConfigFile(themesConfigJsonFile, &cfg); err != nil {
		return cfg, err
	}
	if !isThemeValid(cfg.PortalThemePkg) {
		cfg.PortalThemePkg = defaultThemePlugin
	}
	if !isThemeValid(cfg.AdminThemePkg) {
		cfg.AdminThemePkg = defaultThemePlugin
	}

	themeCfgMu.Lock()
	themeCfgCache = &cfg
	themeCfgMu.Unlock()

	return cfg, nil
}

func WriteThemesConfig(cfg ThemesConfig) error {
	if err := writeConfigFile(themesConfigJsonFile, cfg); err != nil {
		return err
	}

	themeCfgMu.Lock()
	themeCfgCache = &cfg
	themeCfgMu.Unlock()
	return nil
}

func isThemeValid(themePkg string) bool {
	themePath := filepath.Join(sdkutils.PathPluginsDir, themePkg)
	return sdkutils.FsExists(themePath)
}
