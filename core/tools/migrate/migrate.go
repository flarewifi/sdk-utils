package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func MigrateUp(db *sql.DB, pluginDir string) error {
	tmpDir := filepath.Join(sdkutils.PathTmpDir, ".migrate", filepath.Base(pluginDir))
	defer os.RemoveAll(tmpDir)

	files, err := listFiles(pluginDir, tmpDir, migration_Up)
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
				fmt.Printf("Executing migration: %s\n", f)

				if err := execFile(f, tx); err != nil {
					return err
				}

				if err := commitFile(f, tx); err != nil {
					return err
				}

				fmt.Printf("Applied migration: %s\n", f)
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func MigrateDown(plguinDir string, db *sql.DB) error {
	tmpDir := filepath.Join(sdkutils.PathTmpDir, ".migrate", filepath.Base(plguinDir))
	defer os.RemoveAll(tmpDir)

	files, err := listFiles(plguinDir, tmpDir, migration_Down)
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

			fmt.Printf("Reverted migration: %s\n", downfile)

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}
