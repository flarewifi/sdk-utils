package config

import (
	"fmt"
	"path/filepath"
	sdkapi "sdk/api"
	"sync"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const applicationJsonFile = "application.json"

var SupportedLanguages = []sdkapi.SupportedLanguage{
	{Code: "en", Name: "English"},
	{Code: "am", Name: "Amharic"},
	{Code: "ar", Name: "Arabic (Sudan)"},
	{Code: "nl", Name: "Dutch"},
	{Code: "fr", Name: "French"},
	{Code: "hi", Name: "Hindi"},
	{Code: "id", Name: "Indonesian"},
	{Code: "pt", Name: "Portuguese"},
	{Code: "ru", Name: "Russian"},
	{Code: "es", Name: "Spanish"},
	{Code: "vi", Name: "Vietnamese"},
}

var SupportedCurrencies = sdkutils.SupportedCurrencies

var (
	appConfigCache   *sdkapi.AppConfig
	appConfigCacheMu sync.RWMutex
)

// GetCachedAppConfig returns the cached application config.
// If cache is empty, reads from file and caches the result.
func GetCachedAppConfig() (sdkapi.AppConfig, error) {
	appConfigCacheMu.RLock()
	if appConfigCache != nil {
		cfg := *appConfigCache
		appConfigCacheMu.RUnlock()
		return cfg, nil
	}
	appConfigCacheMu.RUnlock()

	// Cache miss - read from file
	appConfigCacheMu.Lock()
	defer appConfigCacheMu.Unlock()

	// Double-check after acquiring write lock
	if appConfigCache != nil {
		return *appConfigCache, nil
	}

	cfg, err := ReadApplicationConfig()
	if err != nil {
		return cfg, err
	}
	appConfigCache = &cfg
	return cfg, nil
}

// updateAppConfigCache updates the cache with new config
func updateAppConfigCache(cfg sdkapi.AppConfig) {
	appConfigCacheMu.Lock()
	defer appConfigCacheMu.Unlock()
	appConfigCache = &cfg
}

var defaultAppCfg = sdkapi.AppConfig{
	Lang:              "en",
	Currency:          "USD",
	Secret:            sdkutils.RandomStr(16),
	Channel:           "stable",
	EnableLogging:     false,
	PluginMaxFileSize: 10 * 1024 * 1024, // 10MB
}

func ReadApplicationConfig() (sdkapi.AppConfig, error) {
	var cfg sdkapi.AppConfig

	err := readConfigFile(applicationJsonFile, &cfg)
	if err != nil {
		// generate defaults if not exists
		fmt.Println(err)
		fmt.Println("Generating default application configuration...")
		defaultFile := filepath.Join(sdkutils.PathDefaultsDir, applicationJsonFile)
		err = writeConfigFile(defaultFile, defaultAppCfg)
		return defaultAppCfg, err
	}

	if cfg.Lang == "" {
		cfg.Lang = defaultAppCfg.Lang
	}

	if cfg.Currency == "" {
		cfg.Currency = defaultAppCfg.Currency
	}

	if cfg.Secret == "" {
		cfg.Secret = defaultAppCfg.Secret
	}

	if cfg.Channel == "" {
		cfg.Channel = defaultAppCfg.Channel
	}

	if cfg.PluginMaxFileSize == 0 {
		cfg.PluginMaxFileSize = defaultAppCfg.PluginMaxFileSize
	}

	return cfg, nil
}

func WriteApplicationConfig(cfg sdkapi.AppConfig) error {
	err := writeConfigFile(applicationJsonFile, cfg)
	if err != nil {
		return err
	}
	updateAppConfigCache(cfg)
	return nil
}
