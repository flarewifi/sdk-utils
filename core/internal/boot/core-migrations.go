package boot

import (
	"log"
	"path/filepath"

	"core/internal/api"
	"core/internal/utils/migrate"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func RunCoreMigrations(g *api.CoreGlobals) {
	db := g.Db.SqlDB()

	err := migrate.Init(db)
	if err != nil {
		log.Println(err)
		return
	}

	err = migrate.MigrateUp(db, filepath.Join(sdkutils.PathCoreDir, "resources/migrations"))
	if err != nil {
		log.Printf("Core migrations error: %s", err.Error())
	} else {
		log.Println("Core migrations success!")
	}
}
