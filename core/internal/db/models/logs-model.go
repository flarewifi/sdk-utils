package models

import (
	"context"
	"core/internal/db"
	"core/internal/db/sqlc"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
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

func NewLogModel(database *db.Database, mdls *Models) *LogModel {
	return &LogModel{
		db:     database,
		models: mdls,
	}
}

func (self *LogModel) Create(ctx context.Context, pkg string, level string, message string, filepath string, line int) error {
	_, err := self.db.Queries.CreateLog(ctx, sqlc.CreateLogParams{
		Package:    pgtype.Text{String: pkg, Valid: pkg != ""},
		Level:      level,
		Message:    message,
		Filepath:   filepath,
		LineNumber: int32(line),
	})
	return err
}

func (self *LogModel) Paginate(ctx context.Context, opts LogsPaginateOpts) (*PaginateResult, error) {

	offset := opts.PerPage * (opts.Page - 1)
	limit := opts.PerPage

	searchOpts := sqlc.SearchLogsParams{
		Offset:     int32(offset),
		Limit:      int32(limit),
		Package:    opts.Package,
		Level:      opts.Level,
		SearchText: opts.SearchText,
	}

	fmt.Printf("Logs filter opts: %+v\n", searchOpts)

	result, err := self.db.Queries.SearchLogs(ctx, searchOpts)
	if err != nil {
		return nil, err
	}

	count, err := self.db.Queries.SearchCount(ctx, sqlc.SearchCountParams{
		Package:    opts.Package,
		Level:      opts.Level,
		SearchText: opts.SearchText,
	})
	if err != nil {
		return nil, err
	}

	logs := make([]*Log, len(result))
	for i, row := range result {
		log := Log{
			Package:   row.Package.String,
			Level:     row.Level,
			Message:   row.Message,
			Filepath:  row.Filepath,
			Line:      int(row.LineNumber),
			CreatedAt: row.CreatedAt.Time,
		}
		logs[i] = &log
	}

	return &PaginateResult{
		Logs:  logs,
		Count: count,
	}, nil
}
