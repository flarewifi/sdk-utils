//go:build dev

package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func BuildQueries(pluginSrc string) error {
	// Generate sqlc files
	sqlcPath := filepath.Join(pluginSrc, "sqlc.yaml")
	migrationsPath := filepath.Join(pluginSrc, "resources/migrations")
	queriesPath := filepath.Join(pluginSrc, "resources/queries")
	// fmt.Println("Checking: ", sqlcPath)

	if sdkutils.FsExists(sqlcPath, migrationsPath, queriesPath) {

		info, err := sdkutils.GetPluginInfoFromPath(pluginSrc)
		if err != nil {
			return err
		}

		workdir := filepath.Join(sdkutils.PathTmpDir, "migrations", info.Package)
		if err := sdkutils.FsEmptyDir(workdir); err != nil {
			return err
		}
		defer os.RemoveAll(workdir)

		if err := sdkutils.FsCopy(sqlcPath, filepath.Join(workdir, "sqlc.yaml")); err != nil {
			return err
		}

		if err := sdkutils.FsCopy(migrationsPath, filepath.Join(workdir, "resources/migrations")); err != nil {
			return err
		}

		if err := sdkutils.FsCopy(queriesPath, filepath.Join(workdir, "resources/queries")); err != nil {
			return err
		}

		coreMigrationsDir := filepath.Join(sdkutils.PathCoreDir, "resources/migrations")
		if err := sdkutils.FsCopy(coreMigrationsDir, filepath.Join(workdir, "resources/migrations")); err != nil {
			return err
		}

		fmt.Println("Running 'sqlc generate' in: ", workdir)
		cmd := exec.Command("sqlc", "generate")
		cmd.Dir = workdir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		resultPath := filepath.Join(workdir, "db/queries")
		if err := sdkutils.FsCopy(resultPath, filepath.Join(pluginSrc, "db/queries")); err != nil {
			return err
		}

		fmt.Println("SQLC generated successfully")
	}

	return nil
}
