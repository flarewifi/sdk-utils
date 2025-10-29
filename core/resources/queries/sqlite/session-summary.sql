-- name: SessionSummary :one
-- engine: sqlite
SELECT
    device_id,
    COUNT(*) AS total_sessions,
    COALESCE(SUM(time_secs), 0) - COALESCE(SUM(consumption_secs), 0) AS remaining_time_secs,
    CAST(COALESCE(SUM(data_mbytes) - SUM(consumption_mb), 0.0) AS REAL) AS remaining_data_mb -- :float
FROM sessions
WHERE
    device_id = $1
    AND (
        (
            session_type = 'time'
            AND consumption_secs < time_secs
        )
        OR (
            session_type = 'data'
            AND consumption_mb < data_mbytes
        )
        OR (
            session_type = 'time-or-data'
            AND consumption_mb < data_mbytes
            AND consumption_secs < time_secs
        )
    )
    AND (
        exp_days IS NULL
        OR started_at IS NULL
        OR (
            exp_days IS NOT NULL
            AND started_at IS NOT NULL
            AND datetime('now') < datetime(started_at, '+' || exp_days || ' days')
        )
    )
GROUP BY device_id;
