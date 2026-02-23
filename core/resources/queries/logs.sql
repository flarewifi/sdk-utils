-- name: CreateLog :one
 INSERT INTO logs (package, level, message, filepath, line_number)
     VALUES (@package, @level, @message, @filepath, @line_number)
 RETURNING
     id;
 
 -- name: ClearLogs :exec
 DELETE FROM logs;
 
 -- name: SearchLogs :many
 SELECT
     *
 FROM
     logs
 WHERE (@package = ''
     OR package = @package)
 AND (@level = ''
     OR level = @level)
 AND (@search_text = ''
     OR LOWER(message)
     LIKE '%' || LOWER(@search_text) || '%')
 ORDER BY
     created_at DESC
 LIMIT @row_limit OFFSET @row_offset;
 
-- name: SearchCount :one
SELECT
    COUNT(id)
FROM
    logs
WHERE (@package = ''
    OR package = @package)
AND (@level = ''
    OR level = @level)
AND (@search_text = ''
    OR LOWER(message)
    LIKE '%' || LOWER(@search_text) || '%');

-- name: CountLogsOlderThan :one
SELECT
    COUNT(id)
FROM
    logs
WHERE
    created_at < datetime('now', '-' || @days || ' days');

-- name: CountAllLogs :one
SELECT
    COUNT(id)
FROM
    logs;

-- name: DeleteLogsOlderThan :exec
DELETE FROM logs
WHERE created_at < datetime('now', '-' || @days || ' days');

 
