-- name: FindAvailableSessionForDevice :one
-- engine: postgresql
-- Note: Elapsed time for running sessions is handled in Go code
SELECT
    *
FROM
  sessions
WHERE
  device_id = @device_id
  AND (
    -- Pure time sessions: check saved consumption only
    (
      session_type = 'time'
      AND time_secs > consumption_secs
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
      AND time_secs > consumption_secs
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
      AND NOW() < started_at + (exp_days * interval '1 day')
    )
  )
LIMIT
  1;


-- name: FindSessionsForDev :many
-- engine: postgresql
-- Note: Elapsed time for running sessions is handled in Go code
SELECT
    *
FROM
  sessions
WHERE
  device_id = @device_id
  AND (
    -- Pure time sessions: check saved consumption only
    (
      session_type = 'time'
      AND time_secs > consumption_secs
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
      AND time_secs > consumption_secs
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
      AND NOW() < started_at + (exp_days * interval '1 day')
    )
  );


-- name: UpdateAllBandwidth :exec
-- engine: postgresql
-- Note: Elapsed time for running sessions is handled in Go code
UPDATE
  sessions
SET
  down_mbits = @down_mbits,
  up_mbits = @up_mbits,
  use_global = @use_global
WHERE
  (
    -- Pure time sessions: check saved consumption only
    (
      session_type = 'time'
      AND time_secs > consumption_secs
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
      AND time_secs > consumption_secs
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
      AND NOW() < started_at + (exp_days * interval '1 day')
    )
  );


-- name: BulkUpdateTimeConsumption :exec
-- engine: postgresql
UPDATE
  sessions
SET
  consumption_secs = consumption_secs + EXTRACT(EPOCH FROM (NOW() - resumed_at))::INTEGER
WHERE
  resumed_at IS NOT NULL;
