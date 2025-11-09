-- name: CreateSession :one
-- engine: sqlite
INSERT INTO sessions (
  device_id, session_type, time_secs,
  data_mbytes, exp_days, down_mbits,
  up_mbits, use_global
)
VALUES
  (@device_id, @session_type, @time_secs, @data_mbytes, @exp_days, @down_mbits, @up_mbits, @use_global) RETURNING id;


-- name: FindSession :one
-- engine: sqlite
SELECT
    *
FROM
  sessions
WHERE
  id = @id
LIMIT
  1;


-- name: UpdateSession :exec
-- engine: sqlite
UPDATE
  sessions
SET
    device_id = @device_id,
    session_type = @session_type,
    time_secs = @time_secs,
    data_mbytes = @data_mbytes,
    consumption_secs = @consumption_secs,
    consumption_mb = @consumption_mb,
    started_at = @started_at,
    exp_days = @exp_days,
    down_mbits = @down_mbits,
    up_mbits = @up_mbits,
    use_global = @use_global
WHERE
  id = @id;


-- name: FindAvailableSessionForDevice :one
-- engine: sqlite
SELECT
    *
FROM
  sessions
WHERE
  device_id = @device_id
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
    (
      exp_days IS NULL
      OR started_at IS NULL
    )
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
