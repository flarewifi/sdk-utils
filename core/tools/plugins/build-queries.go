package plugins

import (
	"core/db"
	"fmt"
	cmd "core/tools/shell"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func BuildQueries(pluginSrc string) error {
	err := cmd.Exec(fmt.Sprintf("./scripts/sqlc-gen.sh %s %s", pluginSrc, db.Driver), &cmd.ExecOpts{
		Dir: sdkutils.PathAppDir,
	})
	return err
}
