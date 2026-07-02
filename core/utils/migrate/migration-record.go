package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
)

// fileDone reports whether a migration file is already recorded as applied.
func fileDone(path string, tx *sql.Tx) (bool, error) {
	var id int32
	q := `SELECT id FROM migrations WHERE file = $1 LIMIT 1`
	err := tx.QueryRow(q, filepath.Base(path)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// commitFile records a migration file as applied.
func commitFile(path string, tx *sql.Tx) error {
	q := `INSERT INTO migrations (file) VALUES ($1)`
	if _, err := tx.Exec(q, filepath.Base(path)); err != nil {
		return fmt.Errorf("failed to record migration file %s: %w", path, err)
	}
	return nil
}

// uncommitFile removes a migration file's applied record.
func uncommitFile(path string, tx *sql.Tx) error {
	q := `DELETE FROM migrations WHERE file = $1`
	if _, err := tx.Exec(q, filepath.Base(path)); err != nil {
		return fmt.Errorf("failed to remove migration record for %s: %w", path, err)
	}
	return nil
}
