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
WHERE (
    s.started_at >= DATETIME('now', 'start of day')
    OR s.resumed_at >= DATETIME('now', 'start of day')
  )
  AND (
    (s.session_type = 'time'          AND s.consumption_secs < s.time_secs)
    OR (s.session_type = 'data'       AND s.consumption_mb < s.data_mbytes)
    OR (s.session_type = 'time-or-data'
        AND s.consumption_secs < s.time_secs
        AND s.consumption_mb < s.data_mbytes)
  )
  AND (
    s.exp_days IS NULL
    OR s.started_at IS NULL
    OR datetime('now') < datetime(s.started_at, '+' || s.exp_days || ' days')
  );

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