-- name: CreateLog :one
INSERT INTO logs (package, level, message, filepath, line_number)
    VALUES (@package, @level, @message, @filepath, @line_number)
RETURNING
    id;

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
LIMIT @limit OFFSET @offset;

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

