package boot

import (
	"fmt"
	"log"

	"core/internal/api"
	"tools/migrate"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func RunCoreMigrations(g *api.CoreGlobals) {
	fmt.Println("Running core migrations...")

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
