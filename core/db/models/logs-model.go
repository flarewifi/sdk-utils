package models

import (
	"context"
	"strconv"

	"core/db"
	"core/db/queries"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

type LogModel struct {
	db     *db.Database
	models *Models
}

type LogsPaginateOpts struct {
	Package    string
	Level      string
	SearchText string
	Page       int
	PerPage    int
}

type PaginateResult struct {
	Logs  []*Log
	Count int64
}

// CreateLogParams holds parameters for creating a new log entry
type CreateLogParams struct {
	Package    string
	Level      string
	Message    string
	Filepath   string
	LineNumber int
}

func NewLogModel(database *db.Database, mdls *Models) *LogModel {
	return &LogModel{
		db:     database,
		models: mdls,
	}
}

func (self *LogModel) Create(ctx context.Context, params CreateLogParams) error {
	_, err := self.db.Queries.CreateLog(ctx, queries.CreateLogParams{
		Package:    sdkutils.StrToNullString(params.Package),
		Level:      params.Level,
		Message:    params.Message,
		Filepath:   params.Filepath,
		LineNumber: int64(params.LineNumber),
	})
	return err
}

func (self *LogModel) Clear(ctx context.Context) error {
	// For SQLite, DELETE is the only truncation option.
	_, err := self.db.DB.ExecContext(ctx, "DELETE FROM logs")
	if err != nil {
		return err
	}

	// Reclaim disk space immediately after a full log clear (user-triggered,
	// so blocking is acceptable). Log any error — don't swallow it silently.


	return nil
}

// CountOlderThan returns the count of logs older than the specified number of days
func (self *LogModel) CountOlderThan(ctx context.Context, days int) (int64, error) {
	daysStr := sdkutils.StrToNullString(strconv.Itoa(days))
	return self.db.Queries.CountLogsOlderThan(ctx, daysStr)
}

// CountAll returns the total count of all logs
func (self *LogModel) CountAll(ctx context.Context) (int64, error) {
	return self.db.Queries.CountAllLogs(ctx)
}

// DeleteOlderThan deletes logs older than the specified number of days
func (self *LogModel) DeleteOlderThan(ctx context.Context, days int) error {
	daysStr := sdkutils.StrToNullString(strconv.Itoa(days))
	err := self.db.Queries.DeleteLogsOlderThan(ctx, daysStr)
	if err != nil {
		return err
	}

	// VACUUM is intentionally omitted — this runs every hour in production and
	// VACUUM in WAL mode would block all concurrent readers/writers each time.
	// WAL checkpointing handles space reclamation automatically.

	return nil
}

// Paginate is implemented in database-specific files:
// - logs-model_sqlite.go
// - logs-model_postgres.go
