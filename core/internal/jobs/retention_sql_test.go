package jobs

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"core/db"
	"core/db/queries"
)

// TestRetentionSQL validates the standardized cleanup retention rules directly
// against the generated queries on an in-memory SQLite: used/consumed resources
// are kept 30 days then deleted; unused/unactivated resources are never deleted
// (only counted for the daily warning); notifications are marked read (stamping
// read_at) and swept 30 days after being read. Timestamps are seeded with SQLite
// datetime() (same format as CURRENT_TIMESTAMP in production) and compared against
// Go-computed cutoffs, mirroring how the jobs run live.
func TestRetentionSQL(t *testing.T) {
	ctx := context.Background()

	sqlDB, err := sql.Open(db.SqliteDriverName, ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	defer sqlDB.Close()

	schema := `
CREATE TABLE sessions (
  id INTEGER PRIMARY KEY,
  session_type VARCHAR(50) NOT NULL DEFAULT 'time',
  time_secs INTEGER NOT NULL DEFAULT 0,
  data_mbytes DECIMAL(18,9) NOT NULL DEFAULT 0,
  consumption_secs INTEGER NOT NULL DEFAULT 0,
  consumption_mb DECIMAL(18,9) NOT NULL DEFAULT 0,
  exp_days INTEGER,
  started_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE vouchers (
  id INTEGER PRIMARY KEY,
  activated_at TIMESTAMP,
  created_at TIMESTAMP
);
CREATE TABLE notifications (
  id INTEGER PRIMARY KEY,
  subject VARCHAR(255) NOT NULL DEFAULT '',
  content TEXT NOT NULL DEFAULT '',
  status INTEGER NOT NULL DEFAULT 0,
  type VARCHAR(100) NOT NULL DEFAULT 'info',
  read_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);`
	if _, err := sqlDB.ExecContext(ctx, schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	q := queries.New(sqlDB)
	const read = int64(1) // sdkapi.NotificationStatusRead

	// --- Sessions: consumed/expired deleted only after 30-day grace ------------
	// S1 consumed 40d ago -> delete; S2 consumed 10d ago -> keep;
	// S3 expired 40d ago (exp_days=1) -> delete; S4 expired 20d ago -> keep;
	// S5 active 40d ago (not consumed, no expiry) -> keep.
	mustExec(t, sqlDB, `INSERT INTO sessions (id, session_type, time_secs, consumption_secs, exp_days, started_at) VALUES
		(1,'time',100,100,NULL,datetime('now','-40 days')),
		(2,'time',100,100,NULL,datetime('now','-10 days')),
		(3,'time',100,0,1,datetime('now','-40 days')),
		(4,'time',100,0,1,datetime('now','-20 days')),
		(5,'time',100,0,NULL,datetime('now','-40 days'))`)

	if got, err := q.CountConsumedOrExpiredSessions(ctx); err != nil || got != 2 {
		t.Fatalf("CountConsumedOrExpiredSessions = %d, err %v, want 2 (S1,S3)", got, err)
	}
	if err := q.DeleteConsumedOrExpiredSessions(ctx); err != nil {
		t.Fatalf("DeleteConsumedOrExpiredSessions: %v", err)
	}
	if remaining := tableIDs(t, sqlDB, "sessions"); !sameSet(remaining, []int64{2, 4, 5}) {
		t.Fatalf("sessions after delete = %v, want [2 4 5]", remaining)
	}

	// --- Vouchers: used deleted after 30d; unused counted (never deleted) ------
	mustExec(t, sqlDB, `INSERT INTO vouchers (id, activated_at, created_at) VALUES
		(1,datetime('now','-40 days'),datetime('now','-45 days')),
		(2,datetime('now','-10 days'),datetime('now','-12 days')),
		(3,NULL,datetime('now','-100 days')),
		(4,NULL,datetime('now','-10 days'))`)

	usedCutoff := sql.NullTime{Time: time.Now().UTC().AddDate(0, 0, -30), Valid: true}
	unusedCutoff := sql.NullTime{Time: time.Now().UTC().AddDate(0, 0, -90), Valid: true}

	if got, err := q.CountUsedVouchers(ctx, usedCutoff); err != nil || got != 1 {
		t.Fatalf("CountUsedVouchers = %d, err %v, want 1 (V1)", got, err)
	}
	if got, err := q.CountUnusedVouchers(ctx, unusedCutoff); err != nil || got != 1 {
		t.Fatalf("CountUnusedVouchers = %d, err %v, want 1 (V3)", got, err)
	}
	if err := q.DeleteUsedVouchers(ctx, usedCutoff); err != nil {
		t.Fatalf("DeleteUsedVouchers: %v", err)
	}
	if remaining := tableIDs(t, sqlDB, "vouchers"); !sameSet(remaining, []int64{2, 3, 4}) {
		t.Fatalf("vouchers after delete = %v, want [2 3 4] (unused V3 must survive)", remaining)
	}

	// --- Notifications: read_at stamping + age sweep --------------------------
	// N1 read 40d ago -> swept; N2 read 10d ago -> keep; N3 unread -> keep.
	mustExec(t, sqlDB, `INSERT INTO notifications (id, status, read_at, created_at) VALUES
		(1,1,datetime('now','-40 days'),datetime('now','-41 days')),
		(2,1,datetime('now','-10 days'),datetime('now','-11 days')),
		(3,0,NULL,datetime('now','-100 days'))`)

	if err := q.DeleteReadNotificationsOlderThan(ctx, queries.DeleteReadNotificationsOlderThanParams{
		Status: read, CutoffDate: usedCutoff,
	}); err != nil {
		t.Fatalf("DeleteReadNotificationsOlderThan: %v", err)
	}
	if remaining := tableIDs(t, sqlDB, "notifications"); !sameSet(remaining, []int64{2, 3}) {
		t.Fatalf("notifications after sweep = %v, want [2 3] (unread N3 must survive)", remaining)
	}

	// UpdateNotificationStatus stamps read_at on read and clears it on unread.
	mustExec(t, sqlDB, `INSERT INTO notifications (id, status) VALUES (4, 0)`)
	if err := q.UpdateNotificationStatus(ctx, queries.UpdateNotificationStatusParams{ID: 4, Status: read}); err != nil {
		t.Fatalf("UpdateNotificationStatus read: %v", err)
	}
	if !readAtSet(t, sqlDB, 4) {
		t.Fatalf("read_at not stamped after marking notification 4 read")
	}
	if err := q.UpdateNotificationStatus(ctx, queries.UpdateNotificationStatusParams{ID: 4, Status: 0}); err != nil {
		t.Fatalf("UpdateNotificationStatus unread: %v", err)
	}
	if readAtSet(t, sqlDB, 4) {
		t.Fatalf("read_at not cleared after marking notification 4 unread")
	}
}

// --- helpers ---------------------------------------------------------------

func mustExec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), q); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

func tableIDs(t *testing.T, db *sql.DB, table string) []int64 {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "SELECT id FROM "+table+" ORDER BY id")
	if err != nil {
		t.Fatalf("select ids from %s: %v", table, err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan id: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

func readAtSet(t *testing.T, db *sql.DB, id int64) bool {
	t.Helper()
	var readAt sql.NullTime
	if err := db.QueryRowContext(context.Background(), "SELECT read_at FROM notifications WHERE id = ?", id).Scan(&readAt); err != nil {
		t.Fatalf("select read_at: %v", err)
	}
	return readAt.Valid
}

func sameSet(got, want []int64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
