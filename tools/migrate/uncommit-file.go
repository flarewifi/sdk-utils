package migrate

import (
	"database/sql"
)

func uncommitFile(path string, tx *sql.Tx) error {
	q := `DELTE FROM migrations WHERE file = "$1" LIMIT 1`
	_, err := tx.Exec(q, path)
	if err != nil {
		return err
	}
	return nil
}
