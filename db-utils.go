package sdkutils

import (
	"context"
	"database/sql"
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
	// Use LevelSerializable so the SQLite driver issues BEGIN IMMEDIATE instead
	// of BEGIN DEFERRED.  IMMEDIATE acquires the write lock upfront, eliminating
	// the read→write upgrade race that causes "database is locked" (SQLITE_BUSY)
	// when multiple goroutines start deferred transactions simultaneously.
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}

	// Always attempt rollback on exit. After a successful Commit() this is a
	// no-op (returns sql.ErrTxDone). This ensures the write lock is released
	// even if fn panics or Commit itself fails.
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}
