package plugins

import (
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

const (
	OutManifestJson = "resources/assets/dist/manifest.json"
)

type CompileResults struct {
	Scripts map[string]string
	Styles  map[string]string
}

type OutputManifest struct {
	AdminAssets  CompileResults `json:"admin"`
	PortalAssets CompileResults `json:"portal"`
	BootAssets   CompileResults `json:"boot"`
}

func GetAssetManifest(pluginDir string) OutputManifest {
	manifestFile := filepath.Join(pluginDir, OutManifestJson)
	emptyRes := CompileResults{
		Scripts: make(map[string]string),
		Styles:  make(map[string]string),
	}
	emptyManifest := OutputManifest{
		AdminAssets:  emptyRes,
		PortalAssets: emptyRes,
		BootAssets:   emptyRes,
	}

	var manifest OutputManifest
	if err := sdkutils.JsonRead(manifestFile, &manifest); err != nil {
		return emptyManifest
	}

	return manifest
}
