-- name: CreateDeviceLog :one
INSERT INTO device_logs (
    device_id,
    message,
    metadata
) VALUES (
    @device_id,
    @message,
    @metadata
) RETURNING id;

-- name: FindDeviceLogsByDeviceID :many
SELECT
    id,
    device_id,
    message,
    metadata,
    created_at
FROM device_logs
WHERE device_id = @device_id
ORDER BY created_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: CountDeviceLogsByDeviceID :one
SELECT COUNT(*) FROM device_logs
WHERE device_id = @device_id;

-- name: DeleteDeviceLogsByDeviceID :exec
DELETE FROM device_logs
WHERE device_id = @device_id;

-- name: DeleteOldDeviceLogs :exec
DELETE FROM device_logs
WHERE created_at < datetime('now', '-90 days');

-- name: DeleteAllDeviceLogs :exec
DELETE FROM device_logs;
