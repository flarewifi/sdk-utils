package migrate

import (
	"path/filepath"
	"sort"
	"strings"

	sdkutils "github.com/flarewifi/sdk-utils"
)

type MigDirection int

const (
	migration_Down MigDirection = iota
	migration_Up
)

// listFiles enumerates migration SQL files for the given plugin directly from
// <pluginDir>/resources/migrations. It deliberately does NOT shell out to any
// build-time helper (e.g. copy-sql.sh) so it works the same in dev and on a
// real router, where the build-time sqlc tooling is not available.
func listFiles(pluginDir string, d MigDirection) (files []string, err error) {
	migrationsDir := filepath.Join(pluginDir, "resources/migrations")

	list := []string{}
	if err = sdkutils.FsListFiles(migrationsDir, &list, false); err != nil {
		return nil, err
	}

	files = []string{}
	if d == migration_Down {
		for _, f := range list {
			if strings.HasSuffix(f, ".down.sql") && !strings.HasPrefix(filepath.Base(f), ".") {
				files = append(files, f)
			}
		}
		sort.Sort(sort.Reverse(sort.StringSlice(files)))
	} else {
		for _, f := range list {
			if strings.HasSuffix(f, ".up.sql") && !strings.HasPrefix(filepath.Base(f), ".") {
				files = append(files, f)
			}
		}
		sort.Strings(files)
	}

	return files, nil
}
