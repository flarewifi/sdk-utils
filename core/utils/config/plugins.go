package config

import (
	jobque "core/utils/job-que"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	readQue  = jobque.NewJobQueue[PluginsConfig]()
	writeQue = jobque.NewJobQueue[struct{}]()
	jsonFile = "plugins.json"
)

type PluginsConfig struct {
	Recompile []string
	Metadata  []sdkutils.PluginMetadata
}

func ReadPluginsConfig() (PluginsConfig, error) {
	empTyCfg := PluginsConfig{Recompile: []string{}, Metadata: []sdkutils.PluginMetadata{}}
	cfg, err := readQue.Exec("ReadPluginsConfig", func() (PluginsConfig, error) {
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
	_, err := writeQue.Exec("WritePluginsConfig", func() (struct{}, error) {
		return struct{}{}, writeConfigFile(jsonFile, cfg)
	})

	return err
}

func ResetPluginsConfig() error {
	cfg := PluginsConfig{Recompile: []string{}, Metadata: []sdkutils.PluginMetadata{}}
	_, err := writeQue.Exec("ResetPluginsConfig", func() (struct{}, error) {
		return struct{}{}, writeConfigFile(jsonFile, cfg)
	})

	return err
}
