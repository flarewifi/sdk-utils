package migrate

import (
	"database/sql"
	"path/filepath"
)

func uncommitFile(path string, tx *sql.Tx) error {
	q := `DELETE FROM migrations WHERE file = $1`
	_, err := tx.Exec(q, filepath.Base(path))
	if err != nil {
		return err
	}
	return nil
}
