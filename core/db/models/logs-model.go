package models

import (
	"context"
	"core/db"
	"core/db/queries"
	"fmt"

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

func NewLogModel(database *db.Database, mdls *Models) *LogModel {
	return &LogModel{
		db:     database,
		models: mdls,
	}
}

func (self *LogModel) Create(ctx context.Context, pkg string, level string, message string, filepath string, line int) error {
	_, err := self.db.Queries.CreateLog(ctx, queries.CreateLogParams{
		Package:    sdkutils.StrToNullString(pkg),
		Level:      level,
		Message:    message,
		Filepath:   filepath,
		LineNumber: int64(line),
	})
	return err
}

func (self *LogModel) Paginate(ctx context.Context, opts LogsPaginateOpts) (*PaginateResult, error) {

	offset := opts.PerPage * (opts.Page - 1)
	limit := opts.PerPage

	searchOpts := queries.SearchLogsParams{
		RowOffset:  int64(offset),
		RowLimit:   int64(limit),
		Package:    opts.Package,
		Level:      opts.Level,
		SearchText: opts.SearchText,
	}

	tx, err := self.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	qtx := self.db.Queries.WithTx(tx)

	fmt.Printf("Logs filter opts: %+v\n", searchOpts)

	result, err := qtx.SearchLogs(ctx, searchOpts)
	if err != nil {
		return nil, err
	}

	count, err := qtx.SearchCount(ctx, queries.SearchCountParams{
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
			CreatedAt: row.CreatedAt,
		}
		logs[i] = &log
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &PaginateResult{
		Logs:  logs,
		Count: count,
	}, nil
}
