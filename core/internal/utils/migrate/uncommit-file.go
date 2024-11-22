package migrate

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func uncommitFile(path string, ctx context.Context, db *pgxpool.Pool) error {
	q := `DELTE FROM migrations WHERE file = "$1" LIMIT 1`
	_, err := db.Exec(ctx, q, path)
	if err != nil {
		return err
	}
	return nil
}
