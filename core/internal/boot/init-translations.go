package boot

import (
	"log"

	"core/utils/config"
	"core/utils/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func InitCoreTranslations() {
	currentLang := "en"
	cfg, err := config.ReadApplicationConfig()
	if err != nil {
		log.Printf("Warning: Failed to read application config for translations, defaulting to 'en': %v", err)
	} else {
		currentLang = cfg.Lang
	}

	log.Printf("[Boot] Initializing translations for language: %s", currentLang)

	// Unpack core translations
	if err := sdkutils.EnsureTranslations(sdkutils.PathCoreDir, currentLang); err != nil {
		log.Printf("Warning: Failed to ensure core translations for language '%s': %v", currentLang, err)
	} else {
		log.Printf("[Boot] Core translations for '%s' initialized successfully.", currentLang)
	}

	// Unpack plugin translations
	for _, dir := range plugins.InstalledPluginDirs() {
		if err := sdkutils.EnsureTranslations(dir, currentLang); err != nil {
			log.Printf("Warning: Failed to ensure translations for plugin dir '%s': %v", dir, err)
		}
	}

	log.Printf("[Boot] Plugin translations for '%s' initialized.", currentLang)
}
