-- name: FindAvailableSessionForDevice :one
-- engine: sqlite
SELECT
    *
FROM
  sessions
WHERE
  device_id = @device_id
    AND (
        -- Pure time sessions: check time accounting for elapsed time if running
        (
            session_type = 'time'
            AND (
                (resumed_at IS NULL AND time_secs - consumption_secs > 0)
                OR
                (resumed_at IS NOT NULL AND time_secs - consumption_secs - CAST((julianday('now') - julianday(resumed_at)) * 86400 AS INTEGER) > 0)
            )
        )
        OR
        -- Pure data sessions: only check data
        (
            session_type = 'data'
            AND consumption_mb < data_mbytes
        )
        OR
        -- Time-or-data sessions: check BOTH time AND data (expires when EITHER runs out)
        (
            session_type = 'time-or-data'
            AND consumption_mb < data_mbytes
            AND (
                (resumed_at IS NULL AND time_secs - consumption_secs > 0)
                OR
                (resumed_at IS NOT NULL AND time_secs - consumption_secs - CAST((julianday('now') - julianday(resumed_at)) * 86400 AS INTEGER) > 0)
            )
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
LIMIT
  1;


-- name: FindSessionsForDev :many
-- engine: sqlite
SELECT
    *
FROM
  sessions
WHERE
  device_id = @device_id
  AND (
    -- Pure time sessions: check time accounting for elapsed time if running
    (
      session_type = 'time'
      AND (
        (resumed_at IS NULL AND time_secs - consumption_secs > 0)
        OR
        (resumed_at IS NOT NULL AND time_secs - consumption_secs - CAST((julianday('now') - julianday(resumed_at)) * 86400 AS INTEGER) > 0)
      )
    )
    OR
    -- Pure data sessions: only check data
    (
      session_type = 'data'
      AND consumption_mb < data_mbytes
    )
    OR
    -- Time-or-data sessions: check BOTH time AND data (expires when EITHER runs out)
    (
      session_type = 'time-or-data'
      AND consumption_mb < data_mbytes
      AND (
        (resumed_at IS NULL AND time_secs - consumption_secs > 0)
        OR
        (resumed_at IS NOT NULL AND time_secs - consumption_secs - CAST((julianday('now') - julianday(resumed_at)) * 86400 AS INTEGER) > 0)
      )
    )
  )
  AND (
    (
      exp_days IS NULL
      OR started_at IS NULL
    )
    OR (
      exp_days IS NOT NULL
      AND started_at IS NOT NULL
      AND datetime('now') < datetime(started_at, '+' || exp_days || ' days')
    )
  );


-- name: UpdateAllBandwidth :exec
-- engine: sqlite
UPDATE
  sessions
SET
  down_mbits = @down_mbits,
  up_mbits = @up_mbits,
  use_global = @use_global
WHERE
  (
    -- Pure time sessions: check time accounting for elapsed time if running
    (
      session_type = 'time'
      AND (
        (resumed_at IS NULL AND time_secs - consumption_secs > 0)
        OR
        (resumed_at IS NOT NULL AND time_secs - consumption_secs - CAST((julianday('now') - julianday(resumed_at)) * 86400 AS INTEGER) > 0)
      )
    )
    OR
    -- Pure data sessions: only check data
    (
      session_type = 'data'
      AND consumption_mb < data_mbytes
    )
    OR
    -- Time-or-data sessions: check BOTH time AND data (expires when EITHER runs out)
    (
      session_type = 'time-or-data'
      AND consumption_mb < data_mbytes
      AND (
        (resumed_at IS NULL AND time_secs - consumption_secs > 0)
        OR
        (resumed_at IS NOT NULL AND time_secs - consumption_secs - CAST((julianday('now') - julianday(resumed_at)) * 86400 AS INTEGER) > 0)
      )
    )
  )
  AND (
    (
      exp_days IS NULL
      OR started_at IS NULL
    )
    OR (
      exp_days IS NOT NULL
      AND started_at IS NOT NULL
      AND datetime('now') < datetime(started_at, '+' || exp_days || ' days')
    )
  );


-- name: BulkUpdateTimeConsumption :exec
-- engine: sqlite
UPDATE
  sessions
SET
  consumption_secs = consumption_secs + CAST((julianday('now') - julianday(resumed_at)) * 86400 AS INTEGER)
WHERE
  resumed_at IS NOT NULL;
