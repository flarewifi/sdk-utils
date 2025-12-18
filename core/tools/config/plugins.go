package config

import (
	jobque "core/tools/job-que"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

var (
	readQue  jobque.JobQue[PluginsConfig]
	writeQue jobque.JobQue[struct{}]
	jsonFile = "plugins.json"
)

type PluginsConfig struct {
	Recompile []string
	Metadata  []sdkutils.PluginMetadata
}

func ReadPluginsConfig() (PluginsConfig, error) {
	empTyCfg := PluginsConfig{Recompile: []string{}, Metadata: []sdkutils.PluginMetadata{}}
	cfg, err := readQue.Exec(func() (PluginsConfig, error) {
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
	_, err := writeQue.Exec(func() (struct{}, error) {
		return struct{}{}, writeConfigFile(jsonFile, cfg)
	})

	return err
}

func ResetPluginsConfig() error {
	cfg := PluginsConfig{Recompile: []string{}, Metadata: []sdkutils.PluginMetadata{}}
	_, err := writeQue.Exec(func() (struct{}, error) {
		return struct{}{}, writeConfigFile(jsonFile, cfg)
	})

	return err
}
