//go:build postgres

package models

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (self *LogModel) Paginate(ctx context.Context, opts LogsPaginateOpts) (*PaginateResult, error) {
	// Build WHERE clause dynamically with PostgreSQL $N placeholders
	whereParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if opts.Package != "" {
		whereParts = append(whereParts, fmt.Sprintf("package = $%d", argIndex))
		args = append(args, opts.Package)
		argIndex++
	}
	if opts.Level != "" {
		whereParts = append(whereParts, fmt.Sprintf("level = $%d", argIndex))
		args = append(args, opts.Level)
		argIndex++
	}
	if opts.SearchText != "" {
		whereParts = append(whereParts, fmt.Sprintf("LOWER(message) LIKE $%d", argIndex))
		args = append(args, "%"+strings.ToLower(opts.SearchText)+"%")
		argIndex++
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

	// Fetch logs with properly numbered parameters
	logQuery := fmt.Sprintf("SELECT id, package, level, message, filepath, line_number, created_at FROM logs%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", whereClause, argIndex, argIndex+1)
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
