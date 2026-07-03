package migrate

import (
	"strings"
	"testing"
)

// nonEmpty trims and drops blank statements, mirroring what execFile skips.
func nonEmpty(stmts []string) []string {
	out := []string{}
	for _, s := range stmts {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func TestSplitStatements(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want int
	}{
		{
			name: "plain statements",
			sql:  "CREATE TABLE a(id INT); CREATE TABLE b(id INT);",
			want: 2,
		},
		{
			// Regression for the notifications read_at migration: a ';' inside a
			// line comment must not split the following statement in two.
			name: "semicolon inside line comment",
			sql: "-- status (0 = unread, 1 = read); read_at is only the timestamp\n" +
				"ALTER TABLE notifications ADD COLUMN read_at TIMESTAMP;\n" +
				"CREATE INDEX idx ON notifications(read_at);",
			want: 2,
		},
		{
			name: "semicolon inside string literal",
			sql:  "INSERT INTO t(v) VALUES ('a;b'); SELECT 1;",
			want: 2,
		},
		{
			name: "semicolon inside block comment",
			sql:  "/* drop; recreate */ CREATE TABLE a(id INT);",
			want: 1,
		},
		{
			name: "trailing statement without terminator",
			sql:  "CREATE TABLE a(id INT);\nCREATE TABLE b(id INT)",
			want: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nonEmpty(splitStatements(tc.sql))
			if len(got) != tc.want {
				t.Fatalf("got %d statements, want %d: %#v", len(got), tc.want, got)
			}
		})
	}
}

// A comment fragment must never survive as a bare (non-comment) SQL statement —
// that was the exact failure mode that rolled back the read_at migration.
func TestSplitStatements_NoLeakedCommentFragment(t *testing.T) {
	sql := "-- read); leaked fragment\nSELECT 1;"
	for _, s := range nonEmpty(splitStatements(sql)) {
		if strings.HasPrefix(s, "leaked fragment") {
			t.Fatalf("comment leaked as SQL fragment: %q", s)
		}
	}
}
