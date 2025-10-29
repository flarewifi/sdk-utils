package config

import (
	"sync"
	jobque "tools/job-que"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	queID    sync.Mutex
	jsonFile = "plugins.json"
)

type PluginsConfig struct {
	Recompile []string
	Metadata  []sdkutils.PluginMetadata
}

func ReadPluginsConfig() (PluginsConfig, error) {
	empTyCfg := PluginsConfig{Recompile: []string{}, Metadata: []sdkutils.PluginMetadata{}}
	cfg, err := jobque.Exec(&queID, func() (PluginsConfig, error) {
		var cfg PluginsConfig
		err := readConfigFile(jsonFile, &cfg)
		if err != nil {
			return empTyCfg, err
		}
		return cfg, nil
	})

	if err != nil {
		return empTyCfg, err
	}

	pluginsCfg := cfg
	if pluginsCfg.Metadata == nil {
		pluginsCfg.Metadata = empTyCfg.Metadata
	}

	return pluginsCfg, nil
}

func WritePluginsConfig(cfg PluginsConfig) error {
	_, err := jobque.Exec(&queID, func() (any, error) {
		return nil, writeConfigFile(jsonFile, cfg)
	})

	return err
}

func ResetPluginsConfig() error {
	cfg := PluginsConfig{Recompile: []string{}, Metadata: []sdkutils.PluginMetadata{}}
	_, err := jobque.Exec(&queID, func() (any, error) {
		return nil, writeConfigFile(jsonFile, cfg)
	})

	return err
}
