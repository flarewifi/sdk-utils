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
-- read_at is stamped when a notification transitions to read (status = 1) and
-- cleared when marked unread, so the cleanup job can age out long-read rows.
UPDATE
  notifications
SET
  status = @status,
  read_at = CASE WHEN @status = 1 THEN datetime('now') ELSE NULL END,
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
-- Only ever marks rows read, so read_at is always stamped to now.
UPDATE
  notifications
SET
  status = @status,
  read_at = datetime('now'),
  updated_at = datetime('now')
WHERE
  status != @status;

-- name: DeleteNotification :exec
DELETE FROM notifications
WHERE
  id = @id;

-- name: DeleteAllNotifications :exec
DELETE FROM notifications;

-- name: DeleteReadNotificationsOlderThan :exec
-- Standardized notification retention: delete notifications that have been READ
-- for more than the retention window. read_at is stamped when a notification is
-- marked read (see UpdateNotificationStatus / MarkAllAsRead); unread rows have a
-- NULL read_at and are never swept here. cutoff_date is computed in Go
-- (time.Now().UTC().AddDate(0, 0, -30)); @status is NotificationStatusRead (1).
DELETE FROM notifications
WHERE status = @status AND read_at IS NOT NULL AND read_at < @cutoff_date;

-- name: DeleteNotificationsExceedingLimit :exec
-- Retention backstop: keeps only the newest 200 notifications and deletes the rest.
-- Read notifications are aged out by DeleteReadNotificationsOlderThan, so this only
-- bounds the UNREAD pile-up (e.g. the daily unused-resource warnings, which are
-- never auto-deleted). The limit is hardcoded here (same pattern as
-- DeleteOldDeviceLogs); id DESC is a tiebreaker for rows sharing a created_at second.
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