-- name: GetConnectedUsersToday :one
SELECT COUNT(DISTINCT d.id)
FROM devices d
WHERE d.status = 1;

-- name: GetSessionsCountToday :one
SELECT COUNT(*) FROM sessions
WHERE created_at >= DATETIME('now', 'start of day')
  AND created_at < DATETIME('now', '+1 day', 'start of day');

-- name: GetAvgSessionSecsToday :one
SELECT COALESCE(AVG(s.consumption_secs), 0)
FROM sessions s
JOIN devices d ON d.id = s.device_id
WHERE d.status = 1
  AND (s.resumed_at IS NOT NULL OR s.started_at IS NOT NULL);

-- name: UpsertPeakUsersToday :exec
INSERT INTO peak_users_today (date, peak_count)
VALUES (DATE('now'), @peak_count)
ON CONFLICT(date) DO UPDATE SET
    peak_count = excluded.peak_count
WHERE excluded.peak_count > peak_users_today.peak_count;

-- name: GetPeakUsersToday :one
SELECT COALESCE(peak_count, 0)
FROM peak_users_today
WHERE date = DATE('now');