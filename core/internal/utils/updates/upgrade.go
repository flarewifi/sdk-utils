package updates

import (
	"core/internal/api"
	"core/internal/utils/plugins"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CompileResult struct {
	Percent int
	Error   error
}

func Preupgrade() error {
	if err := sdkutils.FsCopy(sdkutils.PathConfigDir, filepath.Join(sdkutils.PathSystemUpdateDir, "config")); err != nil {
		return err
	}

	cmd := exec.Command("bin/flare", "upgrade")
	cmd.Dir = sdkutils.PathSystemUpdateDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func UpgradeCmd() error {
	fmt.Println("Running upgrade command (in new version)...")
	g := api.NewGlobals()
	defer g.Db.SqlDB().Close()

	ch := CompilePlugins(g.CoreAPI.SqlDb())

	for result := range ch {
		fmt.Println(result)
	}

	return nil
}

func CompilePlugins(db *pgxpool.Pool) chan CompileResult {
	ch := make(chan CompileResult)

	go func() {
		defer close(ch)
		defs := plugins.AllPluginSrcDefs()
		total := len(defs)

		for i, def := range defs {
			_, err := plugins.InstallSrcDef(os.Stdout, db, def)
			if err != nil {
				result := CompileResult{Error: err}
				ch <- result
				return
			}

			percent := (i + 1) * 100 / total
			result := CompileResult{Percent: percent}
			ch <- result
		}
	}()

	return ch
}
