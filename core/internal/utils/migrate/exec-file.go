package migrate

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func execFile(path string, ctx context.Context, db *pgxpool.Pool) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(b)
	queries := strings.Split(content, ";")

	for _, q := range queries {
		if strings.TrimSpace(q) != "" {
			q = strings.TrimSpace(q)
			_, err = db.Exec(ctx, q)
			if err != nil {
				log.Println(fmt.Sprintf("Error migrating\nfile: %s \n%+v\nquery: %s", path, err, q))
				return fmt.Errorf("error executing query from file %s: %w", path, err)
			}
		}
	}

	return nil
}
