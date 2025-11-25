package boot

import (
	"core/internal/api"
	"core/internal/utils/activation"
	"log"
	"tools/config"
	"tools/env"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func Init(g *api.CoreGlobals) {
	bootCh := make(chan struct{})

	InitDirs()

	go func() {
		g.Database.WaitReady()
		log.Println("Database is ready.")

		// Ensure current language translations are available (decompress if needed)
		if err := initCoreTranslations(); err != nil {
			log.Printf("Warning: Failed to initialize translations: %v", err)
		}

		InitOpkg()
		RunCoreMigrations(g)
		InitPlugins(g)
		InitAssets(g)
		InitAccounts()
		if err := InitNetwork(); err != nil {
			log.Println("Error initializing network:", err)
		}

		// Initialize activation after everything else is ready
		go activation.Validate()

		bootCh <- struct{}{}
	}()

	InitHttpServer(g, bootCh)
}

// initCoreTranslations ensures the current language translations are available
func initCoreTranslations() error {
	// Skip translation decompression in dev mode
	if env.GO_ENV == env.ENV_DEV {
		return nil
	}

	// Read current application config to get the language
	cfg, err := config.ReadApplicationConfig()
	if err != nil {
		return err
	}

	// Ensure the current language is available for core
	return sdkutils.EnsureTranslations(sdkutils.PathCoreDir, cfg.Lang)
}
