-- name: CreateVoucher :one
INSERT INTO vouchers
    (uuid, code, provider_pkg, session_type, time_secs, data_mb, down_speed_mbps, up_speed_mbps, session_exp_days, use_global, expires_at, batch_uuid)
VALUES
    (@uuid, @code, @provider_pkg, @session_type, @time_secs, @data_mb, @down_speed_mbps, @up_speed_mbps, @session_exp_days, @use_global, @expires_at, @batch_uuid)
RETURNING *;

-- name: FindVoucherByCode :one
SELECT *
FROM vouchers
WHERE code = @code
AND activated_at IS NULL
AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
AND provider_pkg = @provider_pkg
LIMIT 1;

-- name: FindVoucherByID :one
SELECT *
FROM vouchers
WHERE id = @id
LIMIT 1;

-- name: GetAllVouchers :many
SELECT *
FROM vouchers
WHERE provider_pkg = @provider_pkg
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: GetAllVouchersCount :one
SELECT COUNT(id)
FROM vouchers
WHERE provider_pkg = @provider_pkg;

-- name: GetVouchersFiltered :many
SELECT v.*
FROM vouchers v
LEFT JOIN devices d ON v.device_id = d.id
WHERE v.provider_pkg = @provider_pkg
AND (
    @search IS NULL OR @search = '' OR
    v.code LIKE '%' || @search || '%' OR
    v.provider_pkg LIKE '%' || @search || '%' OR
    d.mac_address LIKE '%' || @search || '%'
)
AND (
    @is_activated IS NULL OR
    (@is_activated = 1 AND v.activated_at IS NOT NULL) OR
    (@is_activated = 0 AND v.activated_at IS NULL)
)
AND (
    @date_start IS NULL OR v.created_at >= @date_start
    )
AND (
    @date_end IS NULL OR v.created_at <= @date_end
    )
ORDER BY v.created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: GetVouchersFilteredCount :one
SELECT COUNT(v.id)
FROM vouchers v
LEFT JOIN devices d ON v.device_id = d.id
WHERE v.provider_pkg = @provider_pkg
AND (
    @search IS NULL OR @search = '' OR
    v.code LIKE '%' || @search || '%' OR
    v.provider_pkg LIKE '%' || @search || '%' OR
    d.mac_address LIKE '%' || @search || '%'
)
AND (
    @is_activated IS NULL OR
    (@is_activated = 1 AND v.activated_at IS NOT NULL) OR
    (@is_activated = 0 AND v.activated_at IS NULL)
)
AND (
    @date_start IS NULL OR v.created_at >= @date_start
    )
AND (
    @date_end IS NULL OR V.created_at <= @date_end
    );

-- name: UpdateVoucher :exec
UPDATE vouchers
SET session_type = @session_type,
    time_secs = @time_secs,
    data_mb = @data_mb,
    down_speed_mbps = @down_speed_mbps,
    up_speed_mbps = @up_speed_mbps,
    session_exp_days = @session_exp_days,
    use_global = @use_global,
    expires_at = @expires_at
WHERE id = @id;

-- name: ActivateVoucher :exec
UPDATE vouchers
SET session_id = @session_id,
    device_id = @device_id,
    activated_at = CURRENT_TIMESTAMP
WHERE id = @id;

-- name: DeleteVoucherByID :exec
DELETE FROM vouchers
WHERE id = @id;

-- name: DeleteActivatedVouchers :exec
DELETE FROM vouchers
WHERE activated_at IS NOT NULL
AND provider_pkg = @provider_pkg;

-- name: GetAvailableVouchers :many
SELECT *
FROM vouchers
WHERE activated_at IS NULL
AND provider_pkg = @provider_pkg;

-- name: GetActivatedVouchers :many
SELECT *
FROM vouchers
WHERE activated_at IS NOT NULL
AND provider_pkg = @provider_pkg;

-- name: GetVouchersByBatchUUID :many
SELECT *
FROM vouchers
WHERE batch_uuid = @batch_uuid
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: GetVouchersByBatchUUIDCount :one
SELECT COUNT(id)
FROM vouchers
WHERE batch_uuid = @batch_uuid;

-- name: DeleteVouchersByBatchUUID :exec
DELETE FROM vouchers
WHERE batch_uuid = @batch_uuid;
