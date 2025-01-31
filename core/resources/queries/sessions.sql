-- name: CreateSession :one
INSERT INTO sessions (
  device_id, session_type, time_secs, 
  data_mbytes, exp_days, down_mbits, 
  up_mbits, use_global
) 
VALUES 
  ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id;


-- name: FindSession :one
SELECT 
  id, 
  device_id, 
  session_type, 
  time_secs, 
  data_mbytes, 
  consumption_secs, 
  consumption_mb, 
  started_at, 
  exp_days, 
  down_mbits, 
  up_mbits, 
  use_global, 
  created_at, 
    CASE 
        WHEN exp_days IS NOT NULL AND started_at IS NOT NULL
        THEN started_at + INTERVAL '1 day' * exp_days
        ELSE NULL
    END AS expires_at 
FROM 
  sessions 
WHERE 
  id = $1 
LIMIT 
  1;


-- name: UpdateSession :exec
UPDATE 
  sessions 
SET 
  device_id = $1, 
  session_type = $2, 
  time_secs = $3, 
  data_mbytes = $4, 
  consumption_secs = $5, 
  consumption_mb = $6, 
  started_at = $7, 
  exp_days = $8, 
  down_mbits = $9, 
  up_mbits = $10, 
  use_global = $11 
WHERE 
  id = $12;


-- name: FindAvlSessionForDev :one
SELECT 
  id, 
  device_id, 
  session_type, 
  time_secs, 
  data_mbytes, 
  consumption_secs, 
  consumption_mb, 
  started_at, 
  exp_days, 
  down_mbits, 
  up_mbits, 
  use_global, 
  created_at, 
    CASE 
        WHEN exp_days IS NOT NULL AND started_at IS NOT NULL
        THEN started_at + INTERVAL '1 day' * exp_days
        ELSE NULL
    END AS expires_at 
FROM 
  sessions 
WHERE 
  device_id = $1 
  AND (
    (
      session_type = 0 
      AND consumption_secs < time_secs
    ) 
    OR (
      session_type = 1 
      AND consumption_mb < data_mbytes
    ) 
    OR (
      session_type = 2 
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
  id, 
  device_id, 
  session_type, 
  time_secs, 
  data_mbytes, 
  consumption_secs, 
  consumption_mb, 
  started_at, 
  exp_days, 
  down_mbits, 
  up_mbits, 
  use_global, 
  created_at, 
    CASE 
        WHEN exp_days IS NOT NULL AND started_at IS NOT NULL
        THEN started_at + INTERVAL '1 day' * exp_days
        ELSE NULL
    END AS expires_at 
FROM 
  sessions 
WHERE 
  device_id = $1 
  AND (
    (
      session_type = 0 
      AND consumption_secs < time_secs
    ) 
    OR (
      session_type = 1 
      AND consumption_mb < data_mbytes
    ) 
    OR (
      session_type = 2 
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
  down_mbits = $1, 
  up_mbits = $2, 
  use_global = $3 
WHERE 
  (
    (
      session_type = 0 
      AND consumption_secs < time_secs
    ) 
    OR (
      session_type = 1 
      AND consumption_mb < data_mbytes
    ) 
    OR (
      session_type = 2 
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

