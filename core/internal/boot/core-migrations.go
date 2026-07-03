package boot

import (
	"fmt"

	"core/internal/api"
	"core/utils/migrate"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func RunCoreMigrations(g *api.CoreGlobals) {
	if err := migrate.Init(g.Database.DB); err != nil {
		g.CoreAPI.Logger().Error(fmt.Sprintf("boot: core migrations init failed: %v", err))
		return
	}

	if err := migrate.MigrateUp(g.Database.DB, sdkutils.PathCoreDir); err != nil {
		g.CoreAPI.Logger().Error(fmt.Sprintf("boot: core migrations failed: %v", err))
	}
}
