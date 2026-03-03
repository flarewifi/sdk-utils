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


-- name: GetAllSessions :many
SELECT s.* FROM sessions s
ORDER BY s.created_at DESC
LIMIT @row_limit OFFSET @row_offset;


-- name: GetAllSessionsCount :one
SELECT COUNT(*) FROM sessions;


-- name: GetSessionsPaginated :many
SELECT s.* FROM sessions s
LEFT JOIN devices d ON d.id = s.device_id
LEFT JOIN device_macs dm ON d.id = dm.device_id AND dm.is_current = TRUE
LEFT JOIN vouchers v ON v.session_id = s.id
WHERE (
    -- Search filter
    @search IS NULL
    OR s.uuid LIKE '%' || @search || '%'
    OR s.provider_pkg LIKE '%' || @search || '%'
    OR d.uuid LIKE '%' || @search || '%'
    OR dm.mac_address LIKE '%' || @search || '%'
    OR d.hostname LIKE '%' || @search || '%'
    OR d.ip_address LIKE '%' || @search || '%'
    OR v.code LIKE '%' || @search || '%'
)
AND (
    -- Device ID filter
    @device_id IS NULL OR s.device_id = @device_id
)
AND (
    -- Availability filter: 'all', 'available', 'consumed', 'expired'
    @availability = 'all' OR @availability IS NULL OR @availability = ''
    OR (
        -- Available: has remaining time/data AND not expired
        @availability = 'available' AND (
            (s.session_type = 'time' AND s.consumption_secs < s.time_secs)
            OR (s.session_type = 'data' AND s.consumption_mb < s.data_mbytes)
            OR (s.session_type = 'time-or-data' AND s.consumption_secs < s.time_secs AND s.consumption_mb < s.data_mbytes)
        )
        AND (
            s.exp_days IS NULL
            OR s.started_at IS NULL
            OR datetime('now') < datetime(s.started_at, '+' || s.exp_days || ' days')
        )
    )
    OR (
        -- Consumed: time/data exhausted (but not expired by date)
        @availability = 'consumed' AND (
            (s.session_type = 'time' AND s.consumption_secs >= s.time_secs)
            OR (s.session_type = 'data' AND s.consumption_mb >= s.data_mbytes)
            OR (s.session_type = 'time-or-data' AND (s.consumption_secs >= s.time_secs OR s.consumption_mb >= s.data_mbytes))
        )
    )
    OR (
        -- Expired: passed expiration date
        @availability = 'expired' AND (
            s.exp_days IS NOT NULL AND s.started_at IS NOT NULL 
            AND datetime('now') >= datetime(s.started_at, '+' || s.exp_days || ' days')
        )
    )
)
AND (
    -- Session type filter
    @session_type IS NULL OR @session_type = '' OR s.session_type = @session_type
)
AND (
    -- Date range filter: sessions created on or after date_start
    @date_start IS NULL OR date(s.created_at) >= date(@date_start)
)
AND (
    -- Date range filter: sessions created on or before date_end
    @date_end IS NULL OR date(s.created_at) <= date(@date_end)
)
AND (
    -- Time seconds greater than filter
    @time_secs_gt IS NULL OR s.time_secs > @time_secs_gt
)
AND (
    -- Time seconds less than filter
    @time_secs_lt IS NULL OR s.time_secs < @time_secs_lt
)
AND (
    -- Data MB greater than filter
    @data_mb_gt IS NULL OR s.data_mbytes > @data_mb_gt
)
AND (
    -- Data MB less than filter
    @data_mb_lt IS NULL OR s.data_mbytes < @data_mb_lt
)
AND (
    -- Payment type filter: 'all', 'voucher', 'coin'
    @payment_type IS NULL OR @payment_type = '' OR @payment_type = 'all'
    OR (@payment_type = 'voucher' AND v.id IS NOT NULL)
    OR (@payment_type = 'coin' AND v.id IS NULL)
)
ORDER BY s.created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: GetSessionsFiltered :many
SELECT s.* FROM sessions s
LEFT JOIN devices d ON d.id = s.device_id
LEFT JOIN device_macs dm ON d.id = dm.device_id AND dm.is_current = TRUE
LEFT JOIN vouchers v ON v.session_id = s.id
WHERE (
    -- Search filter
    @search IS NULL
    OR s.uuid LIKE '%' || @search || '%'
    OR s.provider_pkg LIKE '%' || @search || '%'
    OR d.uuid LIKE '%' || @search || '%'
    OR dm.mac_address LIKE '%' || @search || '%'
    OR d.hostname LIKE '%' || @search || '%'
    OR d.ip_address LIKE '%' || @search || '%'
    OR v.code LIKE '%' || @search || '%'
)
AND (
    -- Device ID filter
    @device_id IS NULL OR s.device_id = @device_id
)
AND (
    -- Availability filter: 'all', 'available', 'consumed', 'expired'
    @availability = 'all' OR @availability IS NULL OR @availability = ''
    OR (
        -- Available: has remaining time/data AND not expired
        @availability = 'available' AND (
            (s.session_type = 'time' AND s.consumption_secs < s.time_secs)
            OR (s.session_type = 'data' AND s.consumption_mb < s.data_mbytes)
            OR (s.session_type = 'time-or-data' AND s.consumption_secs < s.time_secs AND s.consumption_mb < s.data_mbytes)
        )
        AND (
            s.exp_days IS NULL
            OR s.started_at IS NULL
            OR datetime('now') < datetime(s.started_at, '+' || s.exp_days || ' days')
        )
    )
    OR (
        -- Consumed: time/data exhausted (but not expired by date)
        @availability = 'consumed' AND (
            (s.session_type = 'time' AND s.consumption_secs >= s.time_secs)
            OR (s.session_type = 'data' AND s.consumption_mb >= s.data_mbytes)
            OR (s.session_type = 'time-or-data' AND (s.consumption_secs >= s.time_secs OR s.consumption_mb >= s.data_mbytes))
        )
    )
    OR (
        -- Expired: passed expiration date
        @availability = 'expired' AND (
            s.exp_days IS NOT NULL AND s.started_at IS NOT NULL 
            AND datetime('now') >= datetime(s.started_at, '+' || s.exp_days || ' days')
        )
    )
)
AND (
    -- Session type filter
    @session_type IS NULL OR @session_type = '' OR s.session_type = @session_type
)
AND (
    -- Date range filter: sessions created on or after date_start
    @date_start IS NULL OR date(s.created_at) >= date(@date_start)
)
AND (
    -- Date range filter: sessions created on or before date_end
    @date_end IS NULL OR date(s.created_at) <= date(@date_end)
)
AND (
    -- Time seconds greater than filter
    @time_secs_gt IS NULL OR s.time_secs > @time_secs_gt
)
AND (
    -- Time seconds less than filter
    @time_secs_lt IS NULL OR s.time_secs < @time_secs_lt
)
AND (
    -- Data MB greater than filter
    @data_mb_gt IS NULL OR s.data_mbytes > @data_mb_gt
)
AND (
    -- Data MB less than filter
    @data_mb_lt IS NULL OR s.data_mbytes < @data_mb_lt
)
AND (
    -- Payment type filter: 'all', 'voucher', 'coin'
    @payment_type IS NULL OR @payment_type = '' OR @payment_type = 'all'
    OR (@payment_type = 'voucher' AND v.id IS NOT NULL)
    OR (@payment_type = 'coin' AND v.id IS NULL)
);


-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = @id;

