package migrate

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func commitFile(path string, ctx context.Context, db *pgxpool.Pool) error {
	q := `INSERT INTO migrations (file) VALUES ($1)`
	if _, err := db.Exec(ctx, q, path); err != nil {
		return fmt.Errorf("failed to commit migration file %s: %w", path, err)
	}
	return nil
}
