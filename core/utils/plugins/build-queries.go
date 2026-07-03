package plugins

import (
	cmd "core/utils/shell"
	"fmt"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func BuildQueries(pluginSrc string) error {
	err := cmd.Exec(fmt.Sprintf("./scripts/sqlc-gen.sh %s", pluginSrc), &cmd.ExecOpts{
		Dir: sdkutils.PathAppDir,
	})
	return err
}
