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
    exp_days = @exp_days,
    down_mbits = @down_mbits,
    up_mbits = @up_mbits,
    use_global = @use_global
WHERE
  id = @id;


-- name: ResetAllStartedAt :exec
UPDATE
  sessions
SET
    started_at = NULL
WHERE
    started_at IS NOT NULL;

