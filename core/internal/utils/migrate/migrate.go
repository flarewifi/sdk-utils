package migrate

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func MigrateUp(db *pgxpool.Pool, dir string) error {
	files, err := listFiles(dir, migration_Up)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	// Defer a rollback in case anything fails.
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	for _, f := range files {
		done, err := fileDone(f, ctx, db)
		if err != nil {
			return err
		}

		if !done {
			if err := execFile(f, ctx, db); err != nil {
				return err
			}

			if err := commitFile(f, ctx, db); err != nil {
				return err
			}
		}
	}

	// Commit the transaction.
	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func MigrateDown(dir string, db *pgxpool.Pool) error {
	files, err := listFiles(dir, migration_Down)
	if err != nil {
		return err
	}

	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	// Defer a rollback in case anything fails.
	defer tx.Rollback(ctx)

	for _, downfile := range files {
		upfile := strings.ReplaceAll(downfile, ".down.sql", ".up.sql")
		done, err := fileDone(upfile, ctx, db)
		if err != nil {
			return err
		}

		if done {
			err = execFile(downfile, ctx, db)
			if err != nil {
				return err
			}
			err := uncommitFile(upfile, ctx, db)
			if err != nil {
				return err
			}
		}
	}

	// Commit the transaction.
	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
