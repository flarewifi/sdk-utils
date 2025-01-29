package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func BuildSQLC(pluginSrc string) error {
	// Generate sqlc files
	sqlcPath := filepath.Join(pluginSrc, "sqlc.yaml")
	migrationsPath := filepath.Join(pluginSrc, "resources/migrations")
	queriesPath := filepath.Join(pluginSrc, "resources/queries")
	fmt.Println("Checking: ", sqlcPath)

	if sdkutils.FsExists(sqlcPath, migrationsPath, queriesPath) {
		fmt.Println("Running 'sqlc generate' in: ", pluginSrc)
		cmd := exec.Command("sqlc", "generate")
		cmd.Dir = pluginSrc
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
