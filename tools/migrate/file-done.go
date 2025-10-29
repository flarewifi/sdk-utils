package migrate

import (
	"database/sql"
	"errors"
	"path/filepath"
)

func fileDone(f string, tx *sql.Tx) (exists bool, err error) {

	var id int32
	q := `SELECT id FROM migrations WHERE file = $1 LIMIT 1`

	row := tx.QueryRow(q, filepath.Base(f))
	err = row.Scan(&id)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
