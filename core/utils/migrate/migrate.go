// Package migrate applies core and plugin SQL migrations against the app's
// SQLite database and tracks which files have run in a `migrations` table.
//
// Each plugin (and core) owns a resources/migrations directory of paired
// <timestamp>_<name>.{up,down}.sql files. MigrateUp applies every not-yet-applied
// .up.sql in ascending order; MigrateDown reverts applied migrations in
// descending order. Each file runs in its own transaction, so a failure leaves
// both the schema and the tracking table consistent.
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"

	sdkutils "github.com/flarewifi/sdk-utils"
)

// Init creates the migration tracking table if it does not already exist.
func Init(sqldb *sql.DB) error {
	const q = `CREATE TABLE IF NOT EXISTS migrations (
	    id INTEGER PRIMARY KEY,
	    file VARCHAR(255) NOT NULL,
	    executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := sqldb.Exec(q)
	return err
}

// MigrateUp applies every pending .up.sql migration under
// <pluginDir>/resources/migrations, in ascending filename order. A plugin with
// no migrations directory is a no-op.
func MigrateUp(db *sql.DB, pluginDir string) error {
	files, err := listFiles(pluginDir, directionUp)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	ctx := context.Background()
	for _, f := range files {
		err := sdkutils.RunInTx(db, ctx, func(tx *sql.Tx) error {
			done, err := fileDone(f, tx)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
			if err := execFile(f, tx); err != nil {
				return err
			}
			return commitFile(f, tx)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// MigrateDown reverts applied migrations under <pluginDir>/resources/migrations,
// in descending filename order, running each .down.sql whose paired .up.sql was
// previously applied. A plugin with no migrations directory is a no-op.
func MigrateDown(db *sql.DB, pluginDir string) error {
	files, err := listFiles(pluginDir, directionDown)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	ctx := context.Background()
	for _, downFile := range files {
		upFile := strings.ReplaceAll(downFile, ".down.sql", ".up.sql")
		err := sdkutils.RunInTx(db, ctx, func(tx *sql.Tx) error {
			done, err := fileDone(upFile, tx)
			if err != nil {
				return err
			}
			if !done {
				return nil
			}
			if err := execFile(downFile, tx); err != nil {
				return err
			}
			return uncommitFile(upFile, tx)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
