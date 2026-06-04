package boot

import (
	"core/utils/config"
	"core/utils/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func InitTranslations() {
	currentLang := "en"
	cfg, err := config.ReadApplicationConfig()
	if err == nil {
		currentLang = cfg.Lang
	}

	sdkutils.EnsureTranslations(sdkutils.PathCoreDir, currentLang)

	for _, dir := range plugins.InstalledPluginDirs() {
		sdkutils.EnsureTranslations(dir, currentLang)
	}
}
