-- name: CreateSession :one
INSERT INTO sessions (
  uid, provider_pkg,
  device_id, session_type, time_secs,
  data_mbytes, exp_days, down_mbits,
  up_mbits, use_global
)
VALUES
  (@uid, @provider_pkg, @device_id, @session_type, @time_secs, @data_mbytes, @exp_days, @down_mbits, @up_mbits, @use_global) RETURNING id;


-- name: FindSession :one
SELECT
    *
FROM
  sessions
WHERE
  id = @id
LIMIT
  1;


-- name: UpdateSession :exec
UPDATE
  sessions
SET
    uid = @uid,
    provider_pkg = @provider_pkg,
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
      AND NOW() < started_at + INTERVAL '1 day' * exp_days
    )
  )
LIMIT
  1;


-- name: FindSessionsForDev :many
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
      AND NOW() < started_at + INTERVAL '1 day' * exp_days
    )
  );


-- name: UpdateAllBandwidth :exec
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
      AND NOW() < started_at + INTERVAL '1 day' * exp_days
    )
  );

