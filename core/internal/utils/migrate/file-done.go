package migrate

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func fileDone(f string, ctx context.Context, db *pgxpool.Pool) (exists bool, err error) {
	var id int
	q := `SELECT id FROM migrations WHERE file = $1 LIMIT 1`
	row := db.QueryRow(ctx, q, f)
	err = row.Scan(&id)

	if err != nil && err != pgx.ErrNoRows {
		return false, err
	}

	if err != nil && err == pgx.ErrNoRows {
		return false, nil
	}

	return true, nil
}
