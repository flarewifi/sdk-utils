package updates

import (
	"core/internal/api"
	"core/internal/utils/plugins"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CompileResult struct {
	Percent int
	Error   error
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
			_, err := plugins.InstallSrcDef(db, def)
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
