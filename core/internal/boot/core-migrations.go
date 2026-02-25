package boot

import (
	"log"

	"core/internal/api"
	"core/utils/migrate"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func RunCoreMigrations(g *api.CoreGlobals) {
	err := migrate.Init(g.Database.DB)
	if err != nil {
		log.Println(err)
		return
	}

	err = migrate.MigrateUp(g.Database.DB, sdkutils.PathCoreDir)
	if err != nil {
		log.Printf("Core migrations error: %s", err.Error())
	} else {
		log.Println("Core migrations success!")
	}
}
