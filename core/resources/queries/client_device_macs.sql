-- name: CreateDeviceMac :one
INSERT INTO device_macs (
    device_id,
    mac_address,
    is_current,
    first_seen_at,
    last_seen_at
) VALUES (
    @device_id,
    @mac_address,
    @is_current,
    COALESCE(@first_seen_at, CURRENT_TIMESTAMP),
    COALESCE(@last_seen_at, CURRENT_TIMESTAMP)
) RETURNING id;

-- name: FindMacsByDeviceID :many
SELECT
    id,
    device_id,
    mac_address,
    is_current,
    first_seen_at,
    last_seen_at
FROM device_macs
WHERE device_id = @device_id
ORDER BY last_seen_at DESC;

-- name: FindCurrentMacByDeviceID :one
SELECT
    id,
    device_id,
    mac_address,
    is_current,
    first_seen_at,
    last_seen_at
FROM device_macs
WHERE device_id = @device_id
    AND is_current = TRUE
LIMIT 1;

-- name: CheckExistingMac :one
SELECT
    id,
    device_id,
    mac_address,
    is_current,
    first_seen_at,
    last_seen_at
FROM device_macs
WHERE device_id = @device_id
    AND mac_address = @mac_address
LIMIT 1;

-- name: SetMacAsCurrent :exec
UPDATE device_macs
SET is_current = CASE
    WHEN id = @id THEN TRUE
    ELSE FALSE
END,
last_seen_at = CASE
    WHEN id = @id THEN CURRENT_TIMESTAMP
    ELSE last_seen_at
END
WHERE device_id = @device_id;

-- name: UpdateMacLastSeen :exec
UPDATE device_macs
SET last_seen_at = CURRENT_TIMESTAMP
WHERE id = @id;

-- name: TransferMacs :exec
UPDATE device_macs
SET device_id = @target_device_id
WHERE device_id = @source_device_id;

-- name: GetMacCountByDeviceID :one
SELECT COUNT(*) as count
FROM device_macs
WHERE device_id = @device_id;

-- name: DeleteOldestInactiveMac :exec
DELETE FROM device_macs
WHERE id = (
    SELECT dm.id
    FROM device_macs dm
    WHERE dm.device_id = @device_id
        AND dm.is_current = FALSE
    ORDER BY dm.last_seen_at ASC
    LIMIT 1
);

-- name: FindDeviceByMacAddress :one
SELECT device_id
FROM device_macs
WHERE mac_address = @mac_address
    AND is_current = TRUE
LIMIT 1;

-- name: FindDeviceByAnyMacAddress :one
-- Finds a device by ANY MAC address in history (not just current)
-- Returns the device that most recently used this MAC
SELECT device_id
FROM device_macs
WHERE mac_address = @mac_address
ORDER BY last_seen_at DESC
LIMIT 1;
