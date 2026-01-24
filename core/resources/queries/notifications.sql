-- name: CreateNotification :one
INSERT INTO notifications (
  subject, content, status, type
)
VALUES
  (@subject, @content, @status, @type) RETURNING id;

-- name: GetUnreadNotifications :many
SELECT *
FROM notifications
WHERE status = @status
ORDER BY created_at DESC;

-- name: UpdateNotificationStatus :exec
UPDATE
  notifications
SET
  status = @status,
  updated_at = datetime('now')
WHERE
  id = @id;

-- name: GetByID :one
SELECT *
FROM
  notifications
WHERE
  id = @id
LIMIT
  1;