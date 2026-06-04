package boot

import (
	"core/internal/api"
	"core/utils/migrate"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func RunCoreMigrations(g *api.CoreGlobals) {
	err := migrate.Init(g.Database.DB)
	if err != nil {
		return
	}

	migrate.MigrateUp(g.Database.DB, sdkutils.PathCoreDir)
}
