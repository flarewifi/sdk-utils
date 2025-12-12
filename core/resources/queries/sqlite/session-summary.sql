-- name: SessionSummaryTime :one
-- engine: sqlite
-- Get remaining time from time-based sessions
-- Note: Elapsed time for running sessions is calculated in Go code
SELECT
    CAST(COALESCE(
        SUM(time_secs - consumption_secs), 0
    ) AS INTEGER) AS remaining_time_secs
FROM sessions
WHERE
    device_id = @device_id
    AND session_type IN ('time', 'time-or-data')
    AND time_secs > consumption_secs
    AND (
        -- For time-or-data sessions, also check data hasn't expired
        session_type = 'time'
        OR (session_type = 'time-or-data' AND consumption_mb < data_mbytes)
    )
    AND (
        exp_days IS NULL
        OR started_at IS NULL
        OR (
            exp_days IS NOT NULL
            AND started_at IS NOT NULL
            AND datetime('now') < datetime(started_at, '+' || exp_days || ' days')
        )
    );

-- name: SessionSummaryData :one
-- engine: sqlite
-- Get remaining data from data-based sessions
-- Note: Unsaved data consumption for running sessions is calculated in Go code
SELECT
    CAST(COALESCE(
        SUM(data_mbytes - consumption_mb), 0.0
    ) AS REAL) AS remaining_data_mb -- :float
FROM sessions
WHERE
    device_id = @device_id
    AND session_type IN ('data', 'time-or-data')
    AND consumption_mb < data_mbytes
    AND (
        -- For time-or-data sessions, also check time hasn't expired
        session_type = 'data'
        OR (session_type = 'time-or-data' AND time_secs > consumption_secs)
    )
    AND (
        exp_days IS NULL
        OR started_at IS NULL
        OR (
            exp_days IS NOT NULL
            AND started_at IS NOT NULL
            AND datetime('now') < datetime(started_at, '+' || exp_days || ' days')
        )
    );

-- name: SessionSummaryCount :one
-- engine: sqlite
-- Get total count of active sessions
SELECT
    COUNT(*) AS total_sessions
FROM sessions
WHERE
    device_id = @device_id
    AND (
        (
            session_type = 'time'
            AND (
                CASE
                    WHEN resumed_at IS NOT NULL THEN
                        time_secs - consumption_secs - CAST((julianday('now') - julianday(resumed_at)) * 86400 AS INTEGER)
                    ELSE
                        time_secs - consumption_secs
                END
            ) > 0
        )
        OR (
            session_type = 'data'
            AND consumption_mb < data_mbytes
        )
        OR (
            session_type = 'time-or-data'
            AND consumption_mb < data_mbytes
            AND (
                CASE
                    WHEN resumed_at IS NOT NULL THEN
                        time_secs - consumption_secs - CAST((julianday('now') - julianday(resumed_at)) * 86400 AS INTEGER)
                    ELSE
                        time_secs - consumption_secs
                END
            ) > 0
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
    );
