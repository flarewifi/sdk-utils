-- name: FindAvailableSessionForDevice :one
-- engine: postgresql
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
      AND NOW() < started_at + (exp_days * interval '1 day')
    )
  )
LIMIT
  1;


-- name: FindSessionsForDev :many
-- engine: postgresql
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
      AND NOW() < started_at + (exp_days * interval '1 day')
    )
  );


-- name: UpdateAllBandwidth :exec
-- engine: postgresql
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
      AND NOW() < started_at + (exp_days * interval '1 day')
    )
  );


-- name: BulkUpdateTimeConsumption :exec
-- engine: postgresql
UPDATE
  sessions
SET
  consumption_secs = consumption_secs + EXTRACT(EPOCH FROM (NOW() - started_at))::INTEGER
WHERE
  started_at IS NOT NULL;
