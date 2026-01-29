-- name: CreateSession :one
INSERT INTO sessions (
  uuid, provider_pkg,
  device_id, session_type, time_secs,
  data_mbytes, exp_days, down_mbits,
  up_mbits, use_global
)
VALUES
  (@uuid, @provider_pkg, @device_id, @session_type, @time_secs, @data_mbytes, @exp_days, @down_mbits, @up_mbits, @use_global) RETURNING id;


-- name: FindSession :one
SELECT
    *
FROM
  sessions
WHERE
  id = @id
LIMIT
  1;


-- name: FindSessionByUUID :one
SELECT
    *
FROM
  sessions
WHERE
  uuid = @uuid
LIMIT
  1;


-- name: UpdateSession :exec
UPDATE
  sessions
SET
    provider_pkg = @provider_pkg,
    device_id = @device_id,
    session_type = @session_type,
    time_secs = @time_secs,
    data_mbytes = @data_mbytes,
    consumption_secs = @consumption_secs,
    consumption_mb = @consumption_mb,
    started_at = @started_at,
    resumed_at = @resumed_at,
    exp_days = @exp_days,
    down_mbits = @down_mbits,
    up_mbits = @up_mbits,
    use_global = @use_global
WHERE
  id = @id;


-- name: ResetAllResumedAt :exec
UPDATE
  sessions
SET
    resumed_at = NULL
WHERE
    resumed_at IS NOT NULL;


-- name: FindAvailableSessionForDevice :one
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
        exp_days IS NULL
        OR started_at IS NULL
        OR (
            exp_days IS NOT NULL
            AND started_at IS NOT NULL
            AND datetime('now') < datetime(started_at, '+' || exp_days || ' days')
        )
    )
ORDER BY created_at DESC
LIMIT
  1;


-- name: FindSessionsForDev :many
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
      AND datetime('now') < datetime(started_at, '+' || exp_days || ' days')
    )
  );


-- name: UpdateAllBandwidth :exec
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
      AND datetime('now') < datetime(started_at, '+' || exp_days || ' days')
    )
  );


-- name: BulkUpdateTimeConsumption :exec
UPDATE
  sessions
SET
  consumption_secs = consumption_secs + CAST((julianday('now') - julianday(resumed_at)) * 86400 AS INTEGER)
WHERE
  resumed_at IS NOT NULL;

