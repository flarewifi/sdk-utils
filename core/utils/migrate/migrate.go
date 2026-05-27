package migrate

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func MigrateUp(db *sql.DB, pluginDir string) error {
	files, err := listFiles(pluginDir, migration_Up)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	for _, f := range files {
		ctx := context.Background()
		err := sdkutils.RunInTx(db, ctx, func(tx *sql.Tx) error {
			done, err := fileDone(f, tx)
			if err != nil {
				return err
			}

			if !done {
				if err := execFile(f, tx); err != nil {
					return err
				}

				if err := commitFile(f, tx); err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func MigrateDown(pluginDir string, db *sql.DB) error {
	files, err := listFiles(pluginDir, migration_Down)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	for _, downfile := range files {
		upfile := strings.ReplaceAll(downfile, ".down.sql", ".up.sql")
		ctx := context.Background()

		err := sdkutils.RunInTx(db, ctx, func(tx *sql.Tx) error {
			done, err := fileDone(upfile, tx)
			if err != nil {
				return err
			}

			if done {
				err = execFile(downfile, tx)
				if err != nil {
					return err
				}
				err := uncommitFile(upfile, tx)
				if err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}
