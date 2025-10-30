-- name: CreateNotification :one
INSERT INTO notifications (
  subject, content, status
)
VALUES
  ($1, $2, $3) RETURNING id;

-- name: GetUnreadNotifications :many
SELECT
    *
FROM
  notifications
WHERE
  status = $1;

-- name: UpdateNotificationStatus :exec
UPDATE
  notifications
SET
  status = $1
WHERE
  id = $2;

