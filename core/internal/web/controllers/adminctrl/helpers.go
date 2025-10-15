package adminctrl

import (
	"core/internal/api"
	"encoding/json"
	"errors"
	"log"
	"strings"
)

const (
	PendingStatus = "pending"
	SuccessStatus = "success"
	FailedStatus  = "failed"

	PluginInstallStatusFile = "plugins_install_status"
)

type PluginInstallation struct {
	Source string `json:"source"`
	Status string `json:"status"`
}

func SaveInitialState(api *api.PluginApi, installSource string) error {
	allPlugins := GetAllPlugins(api)

	var found bool
	for i, plugin := range allPlugins {
		if plugin.Source == installSource {
			found = true
			allPlugins[i].Source = installSource
			allPlugins[i].Status = PendingStatus
		}
	}

	if !found {
		p := PluginInstallation{
			Source: installSource,
			Status: PendingStatus,
		}

		allPlugins = append(allPlugins, p)
	}

	b, err := json.Marshal(allPlugins)
	if err != nil {
		return err
	}

	return api.CoreAPI.Config().Plugin().Write(PluginInstallStatusFile, b)
}

func UpdateStatus(api *api.PluginApi, installSource, status string) error {
	allPlugins := GetAllPlugins(api)

	var found bool
	for i, plugin := range allPlugins {
		if plugin.Source == installSource {
			found = true
			allPlugins[i].Status = status
		}
	}

	if !found {
		return errors.New("plugin not found")
	}

	b, err := json.Marshal(allPlugins)
	if err != nil {
		return err
	}

	return api.CoreAPI.Config().Plugin().Write(PluginInstallStatusFile, b)
}

func GetAllPlugins(api *api.PluginApi) []PluginInstallation {
	var plugins []PluginInstallation

	b, err := api.CoreAPI.Config().Plugin().Read(PluginInstallStatusFile)
	if err != nil {
		log.Println("unable to read file: ", err)
		return nil
	}

	if err := json.Unmarshal(b, &plugins); err != nil {
		log.Println("unable to decode: ", err)
		return nil
	}

	return plugins
}

func GetPlugin(api *api.PluginApi, installSource string) *PluginInstallation {
	plugins := GetAllPlugins(api)
	for _, plugin := range plugins {
		if plugin.Source == installSource {
			return &plugin
		}
	}

	return nil
}

func getGithubPluginName(fullURL string) string {
	parts := strings.Split(strings.TrimSuffix(fullURL, "/"), "/")
	return parts[len(parts)-1]
}
