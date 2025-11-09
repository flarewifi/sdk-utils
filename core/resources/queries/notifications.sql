-- name: CreateNotification :one
INSERT INTO notifications (
  subject, content, status
)
VALUES
  (@subject, @content, @status) RETURNING id;

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
  updated_at = CURRENT_TIMESTAMP
WHERE
  id = @id;

