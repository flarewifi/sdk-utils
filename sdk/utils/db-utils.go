package sdkutils

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
)

func StrToFloat64(f string) float64 {
	val, err := strconv.ParseFloat(f, 64)
	if err != nil {
		return 0.0
	}
	return val
}

func Float64ToStr(f float64) string {
	return fmt.Sprintf("%.6f", f)
}

func TimeToNullTime(t *time.Time) sql.NullTime {
	if t != nil {
		return sql.NullTime{Time: *t, Valid: true}
	}
	return sql.NullTime{Valid: false}
}

func Int32ToNullInt32(i *int32) sql.NullInt32 {
	if i != nil {
		return sql.NullInt32{Int32: *i, Valid: true}
	}
	return sql.NullInt32{Valid: false}
}

func StrToNullString(s string) sql.NullString {
	if s != "" {
		return sql.NullString{String: s, Valid: true}
	}
	return sql.NullString{Valid: false}
}

func StrToInt32(id string) int32 {
	val, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		return 0
	}
	return int32(val)
}

func StrToInt64(id string) int64 {
	val, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

func Int64ToNullInt64(i *int64) sql.NullInt64 {
	if i != nil {
		return sql.NullInt64{Int64: *i, Valid: true}
	}
	return sql.NullInt64{Valid: false}
}

func RunInTx(db *sql.DB, ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	err = fn(tx) // Execute the function within the transaction
	if err == nil {
		return tx.Commit()
	}

	rollbackErr := tx.Rollback()
	if rollbackErr != nil {
		return errors.Join(err, rollbackErr)
	}

	return err
}
