package migrate

import (
	"core/db"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	cmd "tools/shell"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type MigDirection int

const (
	migration_Down MigDirection = iota
	migration_Up
)

func listFiles(pluginDir string, tmpdir string, d MigDirection) (files []string, err error) {
	opts := &cmd.ExecOpts{Stdout: os.Stdout, Stderr: os.Stderr}
	if err := cmd.Exec(fmt.Sprintf("./scripts/copy-sql.sh %s %s %s", pluginDir, tmpdir, db.Driver), opts); err != nil {
		return nil, fmt.Errorf("failed to copy migration files: %w", err)
	}

	migrationsDir := filepath.Join(tmpdir, "resources/migrations")

	list := []string{}
	if err = sdkutils.FsListFiles(migrationsDir, &list, false); err != nil {
		return files, err
	}

	files = []string{}
	if d == migration_Down {
		for _, f := range list {
			if strings.HasSuffix(f, ".down.sql") && !strings.HasPrefix(f, ".") {
				files = append(files, f)
			}
		}
		sdkutils.SliceReverseString(files)
	} else {
		for _, f := range list {
			if strings.HasSuffix(f, ".up.sql") && !strings.HasPrefix(f, ".") {
				files = append(files, f)
			}
		}
		sort.Strings(files)
	}

	return files, nil
}
