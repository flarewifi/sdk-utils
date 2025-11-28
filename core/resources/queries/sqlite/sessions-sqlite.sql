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


-- name: BulkUpdateTimeConsumption :exec
-- engine: sqlite
UPDATE
  sessions
SET
  consumption_secs = consumption_secs + CAST((julianday('now') - julianday(started_at)) * 86400 AS INTEGER)
WHERE
  started_at IS NOT NULL;
