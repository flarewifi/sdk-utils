package models

import (
	"context"
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
	_, err := self.db.DB.ExecContext(ctx, "DELETE FROM logs")
	return err
}

// Paginate is implemented in database-specific files:
// - logs-model_sqlite.go
// - logs-model_postgres.go
