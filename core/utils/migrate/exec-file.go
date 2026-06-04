package migrate

import (
	"core/db"
	"database/sql"
	"fmt"
	"strings"
)

func execFile(path string, tx *sql.Tx) error {
	content, err := patchFile(path, db.Driver)
	if err != nil {
		return err
	}

	queries := strings.Split(content, ";")

	for _, q := range queries {
		if strings.TrimSpace(q) != "" {
			q = strings.TrimSpace(q)

			_, err = tx.Exec(q)
			if err != nil {
				return fmt.Errorf("error executing query from file %s: %w", path, err)
			}
		}
	}

	return nil
}
