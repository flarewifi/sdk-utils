package config

import (
	"fmt"
	"path/filepath"
	sdkapi "sdk/api"
	"strings"
	"sync"

	"core/utils/env"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// devCustomDomain is the fixed captive-portal hostname used in local dev, where
// machines carry no per-machine custom_domain. Forcing it here turns on the
// portal scheme split in middlewares.ForceHTTPS (admin over HTTPS, portal over
// HTTP) and funnels portal traffic to this host. Clients resolve it to the dev
// machine via /etc/hosts / Traefik.
const devCustomDomain = "captive.flare-local.com"

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

// HasCustomDomain reports whether a non-empty custom_domain is configured. It is
// the single predicate that gates the machine's "has a cloud-issued portal cert"
// behavior, applied uniformly in dev, staging, and prod:
//
//   - set   => fetch/serve the cloud-issued cert for that domain and force
//     HTTP->HTTPS (funnel portal traffic to the cert-matching host).
//   - empty => serve a self-signed cert and DO NOT force HTTP->HTTPS (there is no
//     cloud cert to funnel to, so forcing TLS would only yield cert warnings).
//
// Consumers: middlewares.ForceHTTPS, httpsserver.ensureTLSCertificates (seed),
// and jobs.performPortalCertFetch (cloud cert fetch).
func HasCustomDomain() bool {
	cfg, err := GetCachedAppConfig()
	if err != nil {
		return false
	}
	return strings.TrimSpace(cfg.CustomDomain) != ""
}

// forceDevCustomDomain pins CustomDomain to devCustomDomain in local dev. Applied
// as config leaves this package (read + write-through cache); never persisted to
// the file, so the on-disk config stays env-agnostic. Prod/staging keep their
// configured value untouched.
func forceDevCustomDomain(cfg sdkapi.AppConfig) sdkapi.AppConfig {
	if env.GO_ENV == env.ENV_DEV {
		cfg.CustomDomain = devCustomDomain
	}
	return cfg
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
		return forceDevCustomDomain(defaultAppCfg), err
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

	return forceDevCustomDomain(cfg), nil
}

func WriteApplicationConfig(cfg sdkapi.AppConfig) error {
	err := writeConfigFile(applicationJsonFile, cfg)
	if err != nil {
		return err
	}
	updateAppConfigCache(forceDevCustomDomain(cfg))
	return nil
}
