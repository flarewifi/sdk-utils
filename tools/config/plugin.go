package config

import (
	"os"
	"path/filepath"
	sdkapi "sdk/api"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func NewPluginCfgApi(pkg string) *PluginCfgApi {
	dirPath := filepath.Join(sdkutils.PathConfigDir, "plugins", pkg)
	return &PluginCfgApi{
		Package:       pkg,
		PluginCfgPath: dirPath,
	}
}

type PluginCfgApi struct {
	Package       string
	PluginCfgPath string
}

func (p *PluginCfgApi) Read(key string) ([]byte, error) {
	file := filepath.Join(p.PluginCfgPath, key)
	return os.ReadFile(file)
}

func (p *PluginCfgApi) Write(key string, data []byte) error {
	file := filepath.Join(p.PluginCfgPath, key)
	return sdkutils.FsWriteFile(file, data)
}

func (p *PluginCfgApi) List(path string) ([]*sdkapi.ConfigEntry, error) {
	entries, err := os.ReadDir(filepath.Join(p.PluginCfgPath, path))
	if err != nil {
		return nil, err
	}

	files := make([]*sdkapi.ConfigEntry, len(entries))
	for i, entry := range entries {
		fullpath := filepath.Join(path, entry.Name())
		files[i] = &sdkapi.ConfigEntry{
			Entry: entry,
			Path:  fullpath,
		}
	}

	return files, nil
}

func (p *PluginCfgApi) Delete(path string) error {
	file := filepath.Join(p.PluginCfgPath, path)
	return os.RemoveAll(file)
}
