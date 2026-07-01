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

-- name: MarkAllAsRead :exec
UPDATE
  notifications
SET
  status = @status,
  updated_at = datetime('now')
WHERE
  status != @status;

-- name: DeleteNotification :exec
DELETE FROM notifications
WHERE
  id = @id;

-- name: DeleteAllNotifications :exec
DELETE FROM notifications;

-- name: DeleteNotificationsExceedingLimit :exec
-- Retention cap: keeps only the newest 200 notifications and deletes the rest.
-- Read notifications are already deleted on open, so this bounds the UNREAD
-- pile-up (including this subsystem's own nightly warnings). The limit is
-- hardcoded here (same pattern as DeleteOldDeviceLogs); id DESC is a tiebreaker
-- for rows sharing a created_at second.
DELETE FROM notifications
WHERE id NOT IN (
  SELECT id FROM notifications
  ORDER BY created_at DESC, id DESC
  LIMIT 200
);

-- name: CountRecentNotificationsBySubject :one
-- Counts notifications with the same subject created on/after cutoff_date. Used to
-- throttle repeat warnings so a persistent condition doesn't re-notify every night.
SELECT COUNT(*) FROM notifications
WHERE subject = @subject AND created_at >= @cutoff_date;