package models

import (
	"context"
	"core/db"
	"core/db/queries"
	"strings"
	"time"

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

func (self *LogModel) Paginate(ctx context.Context, opts LogsPaginateOpts) (*PaginateResult, error) {
	// Build WHERE clause dynamically
	whereParts := []string{}
	args := []interface{}{}
	if opts.Package != "" {
		whereParts = append(whereParts, "package = ?")
		args = append(args, opts.Package)
	}
	if opts.Level != "" {
		whereParts = append(whereParts, "level = ?")
		args = append(args, opts.Level)
	}
	if opts.SearchText != "" {
		whereParts = append(whereParts, "LOWER(message) LIKE ?")
		args = append(args, "%"+strings.ToLower(opts.SearchText)+"%")
	}
	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = " WHERE " + strings.Join(whereParts, " AND ")
	}

	offset := int64(opts.PerPage * (opts.Page - 1))
	limit := int64(opts.PerPage)

	// Count
	countQuery := "SELECT COUNT(id) FROM logs" + whereClause
	var count int64
	if err := self.db.DB.QueryRowContext(ctx, countQuery, args...).Scan(&count); err != nil {
		return nil, err
	}

	// Fetch logs
	logQuery := "SELECT id, package, level, message, filepath, line_number, created_at FROM logs" + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	qArgs := append(args, limit, offset)
	rows, err := self.db.DB.QueryContext(ctx, logQuery, qArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]*Log, 0, limit)
	for rows.Next() {
		var l Log
		var id int64
		var pkg, lvl, msg, path string
		var line int64
		var createdAt time.Time
		if err := rows.Scan(&id, &pkg, &lvl, &msg, &path, &line, &createdAt); err != nil {
			return nil, err
		}
		l = Log{Package: pkg, Level: lvl, Message: msg, Filepath: path, Line: int(line), CreatedAt: createdAt}
		logs = append(logs, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &PaginateResult{Logs: logs, Count: count}, nil
}
