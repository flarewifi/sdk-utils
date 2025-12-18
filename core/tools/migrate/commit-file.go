package migrate

import (
	"database/sql"
	"fmt"
	"path/filepath"
)

func commitFile(path string, tx *sql.Tx) error {
	q := `INSERT INTO migrations (file) VALUES ($1)`
	if _, err := tx.Exec(q, filepath.Base(path)); err != nil {
		return fmt.Errorf("failed to commit migration file %s: %w", path, err)
	}
	return nil
}
