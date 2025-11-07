-- name: CreateNotification :one
INSERT INTO notifications (
  subject, content, status
)
VALUES
  ($1, $2, $3) RETURNING id;

-- name: GetUnreadNotifications :many
SELECT *
FROM notifications
WHERE status = $1
ORDER BY created_at DESC;

-- name: UpdateNotificationStatus :exec
UPDATE
  notifications
SET
  status = $1,
  updated_at = CURRENT_TIMESTAMP
WHERE
  id = $2;

