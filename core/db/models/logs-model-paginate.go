package models

import (
	"context"
	"strings"
	"time"
)

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
